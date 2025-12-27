package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"messaging-app/internal/kafka"
	"messaging-app/internal/repositories"
	"messaging-app/internal/userclient"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/events"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	pb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"

	"github.com/redis/go-redis/v9"
	kafkago "github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo      *repositories.UserRepository
	reelRepo      *repositories.ReelRepository
	redisClient   *redis.ClusterClient
	feedService   *FeedService
	kafkaProducer *kafka.MessageProducer

	// New gRPC Client
	grpcClient *userclient.Client
}

func NewUserService(
	userRepo *repositories.UserRepository,
	reelRepo *repositories.ReelRepository,
	redisClient *redis.ClusterClient,
	feedService *FeedService,
	kafkaProducer *kafka.MessageProducer,
	grpcClient *userclient.Client, // Inject client
) *UserService {
	return &UserService{
		userRepo:      userRepo,
		reelRepo:      reelRepo,
		redisClient:   redisClient,
		feedService:   feedService,
		kafkaProducer: kafkaProducer,
		grpcClient:    grpcClient,
	}
}

// UserUpdatedEvent represents a user profile update event
type UserUpdatedEvent struct {
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	FullName    string     `json:"full_name"`
	Avatar      string     `json:"avatar"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
}

func (s *UserService) GetUserStatus(ctx context.Context, userID primitive.ObjectID) (map[string]interface{}, error) {
	// ... (Keep existing implementation for Redis presence)
	key := fmt.Sprintf("presence:%s", userID.Hex())
	val, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return map[string]interface{}{"status": "offline"}, nil
		}
		return nil, err
	}

	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(val), &statusData); err != nil {
		return nil, err
	}

	return statusData, nil
}

// GetUserByID now with Read-Through Caching
func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	// 1. Try Cache
	cacheKey := fmt.Sprintf("user:profile:%s", id.Hex())
	val, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var user models.User
		if err := json.Unmarshal([]byte(val), &user); err == nil {
			return &user, nil
		}
	}

	var user *models.User
	// 2. Try gRPC
	if s.grpcClient != nil {
		user, err = s.grpcClient.GetUserByID(ctx, id)
		if err != nil {
			// Fail "safe" - fallback to local repo if gRPC fails (Robustness)
			log.Printf("gRPC GetUserByID failed, falling back to local repo: %v", err)
			user, err = s.userRepo.FindUserByID(ctx, id)
		}
	} else {
		// Fallback to local repo
		user, err = s.userRepo.FindUserByID(ctx, id)
	}

	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// 3. Set Cache (Async)
	go func(u *models.User) {
		bytes, _ := json.Marshal(u)
		s.redisClient.Set(context.Background(), cacheKey, bytes, time.Hour*24) // 24h TTL, relies on Event Invalidation
	}(user)

	return user, nil
}

// GetUsersByIDs - Batch with Caching (MGET Optimized)
func (s *UserService) GetUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}

	users := make([]models.User, 0, len(ids))
	missIDs := make([]primitive.ObjectID, 0)

	// Deduplicate IDs to avoid redundant work
	idSet := make(map[string]struct{})
	uniqueIDs := make([]primitive.ObjectID, 0, len(ids))

	for _, id := range ids {
		hex := id.Hex()
		if _, exists := idSet[hex]; !exists {
			idSet[hex] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// 1. Check Cache with MGET (O(1) RTT)
	keys := make([]string, len(uniqueIDs))
	for i, id := range uniqueIDs {
		keys[i] = fmt.Sprintf("user:profile:%s", id.Hex())
	}

	vals, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		// Log error and fallback to fetching all from source
		log.Printf("Redis MGET failed: %v", err)
		missIDs = uniqueIDs
	} else {
		for i, val := range vals {
			if val == nil {
				missIDs = append(missIDs, uniqueIDs[i])
				continue
			}

			var user models.User
			if valStr, ok := val.(string); ok {
				if err := json.Unmarshal([]byte(valStr), &user); err == nil {
					users = append(users, user)
					continue
				}
			}
			// If invalid or error, treat as miss
			missIDs = append(missIDs, uniqueIDs[i])
		}
	}

	if len(missIDs) == 0 {
		return users, nil
	}

	// 2. Fetch Misses from Source
	var fetchedUsers []models.User
	if s.grpcClient != nil {
		fetchedUsers, err = s.grpcClient.GetUsersByIDs(ctx, missIDs)
		if err != nil {
			log.Printf("gRPC GetUsersByIDs failed, falling back to local repo: %v", err)
			fetchedUsers, err = s.userRepo.FindUsersByIDs(ctx, missIDs)
		}
	} else {
		fetchedUsers, err = s.userRepo.FindUsersByIDs(ctx, missIDs)
	}

	if err != nil {
		return nil, err
	}

	// 3. Cache Misses & Combine (Pipeline SET)
	if len(fetchedUsers) > 0 {
		pipe := s.redisClient.Pipeline()
		for _, u := range fetchedUsers {
			users = append(users, u)

			bytes, _ := json.Marshal(u)
			cacheKey := fmt.Sprintf("user:profile:%s", u.ID.Hex())
			pipe.Set(ctx, cacheKey, bytes, time.Hour*24)
		}

		// Exec non-blocking or blocking?
		// Context is critical. If we want this to be async, we launch a goroutine.
		// Async is better for user latency.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := pipe.Exec(ctx); err != nil {
				log.Printf("Failed to pipeline cache updates: %v", err)
			}
		}()
	}

	return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, update *models.UserUpdateRequest) (*models.User, error) {
	updateData := bson.M{
		"updated_at": time.Now(),
	}

	if update.Username != "" {
		existingUserUsername, _ := s.userRepo.FindUserByUserName(ctx, update.Username)
		if existingUserUsername != nil && existingUserUsername.ID != id {
			return nil, errors.New("username already exists")
		}
		updateData["username"] = update.Username
	}

	if update.Email != "" {
		existingUserEmail, _ := s.userRepo.FindUserByEmail(ctx, update.Email)
		if existingUserEmail != nil && existingUserEmail.ID != id {
			return nil, errors.New("user email already exists")
		}
		updateData["email"] = update.Email
		updateData["email_verified"] = false // Email needs re-verification
	}

	// Only update password if new password provided
	if update.CurrentPassword != "" && update.NewPassword != "" {
		user, err := s.userRepo.FindUserByID(ctx, id)
		if err != nil {
			return nil, err
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(update.CurrentPassword)); err != nil {
			return nil, errors.New("current password is incorrect")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(update.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updateData["password"] = string(hashedPassword)
	}

	if update.FullName != "" {
		updateData["full_name"] = update.FullName
	}
	if update.Bio != "" {
		updateData["bio"] = update.Bio
	}
	if update.DateOfBirth != nil {
		updateData["date_of_birth"] = update.DateOfBirth
	}
	if update.Gender != "" {
		updateData["gender"] = update.Gender
	}
	if update.Location != "" {
		updateData["location"] = update.Location
	}
	if update.PhoneNumber != "" {
		updateData["phone_number"] = update.PhoneNumber
	}
	if update.Avatar != "" {
		updateData["avatar"] = update.Avatar

		// Create Post and Add to Album
		go func(newAvatar string) {
			// Create Post
			postReq := &models.CreatePostRequest{
				Content: "Updated their profile picture.",
				Media: []models.MediaItem{{
					Type: "image",
					URL:  newAvatar,
				}},
				Privacy: models.PrivacySettingPublic,
			}
			s.feedService.CreatePost(context.Background(), id, postReq)

			// Add to Album
			album, err := s.feedService.EnsureAlbumExists(context.Background(), id, models.AlbumTypeProfile, "Profile Pictures", true)
			if err == nil && album != nil {
				s.feedService.AddMediaToAlbum(context.Background(), id, album.ID, postReq.Media)
			}
		}(update.Avatar)
	}
	if update.CoverPicture != "" {
		updateData["cover_picture"] = update.CoverPicture

		// Create Post and Add to Album
		go func(newCover string) {
			// Create Post
			postReq := &models.CreatePostRequest{
				Content: "Updated their cover photo.",
				Media: []models.MediaItem{{
					Type: "image",
					URL:  newCover,
				}},
				Privacy: models.PrivacySettingPublic,
			}
			s.feedService.CreatePost(context.Background(), id, postReq)

			// Add to Album
			album, err := s.feedService.EnsureAlbumExists(context.Background(), id, models.AlbumTypeCover, "Cover Photos", true)
			if err == nil && album != nil {
				s.feedService.AddMediaToAlbum(context.Background(), id, album.ID, postReq.Media)
			}
		}(update.CoverPicture)
	}
	if update.IsEncryptionEnabled != nil {
		updateData["is_encryption_enabled"] = *update.IsEncryptionEnabled
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, id, updateData)
	if err != nil {
		return nil, err
	}

	// Invalidate user profile cache
	cacheKey := fmt.Sprintf("user:profile:%s", id.Hex())
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate user cache for %s: %v", id.Hex(), err)
	} else {
		log.Printf("Invalidated user cache for %s", id.Hex())
	}

	// Publish UserUpdated event
	if s.kafkaProducer != nil {
		event := events.UserUpdatedEvent{
			UserID:      updatedUser.ID.Hex(),
			Username:    updatedUser.Username,
			FullName:    updatedUser.FullName,
			Avatar:      updatedUser.Avatar,
			DateOfBirth: updatedUser.DateOfBirth,
		}

		eventBytes, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal UserUpdatedEvent: %v", err)
		} else {
			kafkaMsg := kafkago.Message{
				Key:   []byte(updatedUser.ID.Hex()),
				Value: eventBytes,
				Time:  time.Now(),
			}
			if err := s.kafkaProducer.ProduceMessage(ctx, kafkaMsg); err != nil {
				log.Printf("Failed to produce UserUpdatedEvent: %v", err)
			} else {
				log.Printf("Published UserUpdatedEvent for user %s", updatedUser.ID.Hex())
			}
		}
	}

	// Clear password before returning
	updatedUser.Password = ""

	// Sync author info to Reels (async or sync? Sync for now to ensure consistency)
	if update.Username != "" || update.Avatar != "" || update.FullName != "" {
		go func() {
			author := models.PostAuthor{
				ID:       updatedUser.ID.Hex(),
				Username: updatedUser.Username,
				Avatar:   updatedUser.Avatar,
				FullName: updatedUser.FullName,
			}
			// We use a background context or a new one
			if err := s.reelRepo.UpdateAuthorInfo(context.Background(), id, author); err != nil {
				log.Printf("Failed to sync author info to reels: %v", err)
			}
		}()
	}

	return updatedUser, nil
}

// UpdateNotificationSettings updates a user's notification settings via gRPC
func (s *UserService) UpdateNotificationSettings(ctx context.Context, userID primitive.ObjectID, req *models.UpdateNotificationSettingsRequest) error {
	grpcReq := &pb.UpdateNotificationSettingsRequest{
		UserId: userID.Hex(),
	}

	if req.EmailNotifications != nil {
		grpcReq.EmailNotifications = req.EmailNotifications
	}
	if req.PushNotifications != nil {
		grpcReq.PushNotifications = req.PushNotifications
	}
	if req.NotifyOnFriendRequest != nil {
		grpcReq.NotifyOnFriendRequest = req.NotifyOnFriendRequest
	}
	if req.NotifyOnComment != nil {
		grpcReq.NotifyOnComment = req.NotifyOnComment
	}
	if req.NotifyOnLike != nil {
		grpcReq.NotifyOnLike = req.NotifyOnLike
	}
	if req.NotifyOnTag != nil {
		grpcReq.NotifyOnTag = req.NotifyOnTag
	}
	if req.NotifyOnMessage != nil {
		grpcReq.NotifyOnMessage = req.NotifyOnMessage
	}

	_, err := s.grpcClient.UpdateNotificationSettings(ctx, grpcReq)
	if err != nil {
		return fmt.Errorf("failed to update notification settings via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}

// ListUsers retrieves a paginated list of users via gRPC
func (s *UserService) ListUsers(ctx context.Context, page, limit int64, search string) (*models.UserListResponse, error) {
	resp, err := s.grpcClient.ListUsers(ctx, page, limit, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list users via gRPC: %w", err)
	}

	// Convert proto users to models
	users := make([]models.User, len(resp.Users))
	for i, pu := range resp.Users {
		oid, _ := primitive.ObjectIDFromHex(pu.Id)
		users[i] = models.User{
			ID:       oid,
			Username: pu.Username,
			Email:    pu.Email,
			FullName: pu.FullName,
			Avatar:   pu.Avatar,
			Bio:      pu.Bio,
			IsActive: pu.IsActive,
		}
	}

	return &models.UserListResponse{
		Users: users,
		Total: resp.Total,
		Page:  resp.Page,
		Limit: resp.Limit,
	}, nil
}

// UpdateEmail updates a user's email address via gRPC
func (s *UserService) UpdateEmail(ctx context.Context, userID primitive.ObjectID, req *models.UpdateEmailRequest) error {
	_, err := s.grpcClient.UpdateEmail(ctx, userID.Hex(), req.NewEmail)
	if err != nil {
		return fmt.Errorf("failed to update email via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}

// UpdatePassword updates a user's password via gRPC
func (s *UserService) UpdatePassword(ctx context.Context, userID primitive.ObjectID, req *models.UpdatePasswordRequest) error {
	_, err := s.grpcClient.UpdatePassword(ctx, userID.Hex(), req.CurrentPassword, req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to update password via gRPC: %w", err)
	}

	return nil
}

// ToggleTwoFactor enables or disables two-factor authentication via gRPC
func (s *UserService) ToggleTwoFactor(ctx context.Context, userID primitive.ObjectID, enable bool) error {
	_, err := s.grpcClient.ToggleTwoFactor(ctx, userID.Hex(), enable)
	if err != nil {
		return fmt.Errorf("failed to toggle two-factor authentication via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}

// DeactivateAccount deactivates a user's account via gRPC
func (s *UserService) DeactivateAccount(ctx context.Context, userID primitive.ObjectID) error {
	_, err := s.grpcClient.DeactivateAccount(ctx, userID.Hex())
	if err != nil {
		return fmt.Errorf("failed to deactivate account via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}

// UpdatePublicKey updates a user's E2EE public key via gRPC
func (s *UserService) UpdatePublicKey(ctx context.Context, userID primitive.ObjectID, publicKey, encryptedPrivateKey, iv, salt string) error {
	_, err := s.grpcClient.UpdatePublicKey(ctx, userID.Hex(), publicKey, encryptedPrivateKey, iv, salt)
	if err != nil {
		return fmt.Errorf("failed to update keys via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}

// GetUsersPresence retrieves the presence status for a list of user IDs
func (s *UserService) GetUsersPresence(ctx context.Context, userIDs []primitive.ObjectID) (map[string]map[string]interface{}, error) {
	presenceMap := make(map[string]map[string]interface{})

	keys := make([]string, len(userIDs))
	for i, userID := range userIDs {
		keys[i] = fmt.Sprintf("presence:%s", userID.Hex())
	}

	// MGET all presence keys from Redis
	vals, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get presence from Redis: %w", err)
	}

	for i, userID := range userIDs {
		var statusData map[string]interface{}
		val := vals[i]

		if val == nil {
			// Key not found, assume offline
			statusData = map[string]interface{}{"status": "offline", "last_seen": time.Now().Unix()}
		} else {
			if err := json.Unmarshal([]byte(val.(string)), &statusData); err != nil {
				// Error unmarshaling, assume offline and log error
				log.Printf("Error unmarshaling presence data for user %s: %v", userID.Hex(), err)
				statusData = map[string]interface{}{"status": "offline", "last_seen": time.Now().Unix()}
			}
		}
		presenceMap[userID.Hex()] = statusData
	}

	return presenceMap, nil
}

// UpdatePrivacySettings updates a user's privacy settings via gRPC
func (s *UserService) UpdatePrivacySettings(ctx context.Context, userID primitive.ObjectID, req *models.UpdatePrivacySettingsRequest) error {
	settings := map[string]string{}
	if req.DefaultPostPrivacy != "" {
		settings["default_post_privacy"] = string(req.DefaultPostPrivacy)
	}
	if req.CanSeeMyFriendsList != "" {
		settings["can_see_my_friends_list"] = string(req.CanSeeMyFriendsList)
	}
	if req.CanSendMeFriendRequests != "" {
		settings["can_send_me_friend_requests"] = string(req.CanSendMeFriendRequests)
	}
	if req.CanTagMeInPosts != "" {
		settings["can_tag_me_in_posts"] = string(req.CanTagMeInPosts)
	}

	if len(settings) == 0 {
		return nil // No updates
	}

	_, err := s.grpcClient.UpdatePrivacySettings(ctx, userID.Hex(), settings)
	if err != nil {
		return fmt.Errorf("failed to update privacy settings via gRPC: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user_profile:%s", userID.Hex())
	s.redisClient.Del(ctx, cacheKey)

	return nil
}
