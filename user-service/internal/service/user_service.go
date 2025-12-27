package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"user-service/config"
	"user-service/internal/platform"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo    UserRepository
	producer    EventProducer
	redisClient redis.UniversalClient
	cfg         *config.Config
	logger      *slog.Logger
	metrics     *platform.BusinessMetrics
}

func NewUserService(userRepo UserRepository, producer EventProducer, redisClient redis.UniversalClient, cfg *config.Config, logger *slog.Logger, metrics *platform.BusinessMetrics) *UserService {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserService{
		userRepo:    userRepo,
		producer:    producer,
		redisClient: redisClient,
		cfg:         cfg,
		logger:      logger,
		metrics:     metrics,
	}
}

// publishUserUpdatedEvent publishes an event to Kafka when user data changes
func (s *UserService) publishUserUpdatedEvent(ctx context.Context, userID string, updatedUser *models.User) {
	event := map[string]interface{}{
		"event_type": "USER_UPDATED",
		"user_id":    userID,
		"timestamp":  time.Now(),
		"user_data":  updatedUser,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal user event", "user_id", userID, "error", err)
		return
	}

	// Synchronous publish with retry (handled by producer)
	if err := s.producer.Produce(ctx, []byte(userID), payload); err != nil {
		s.logger.Error("Failed to publish USER_UPDATED event", "user_id", userID, "error", err)
		return
	}

	s.logger.Info("Published USER_UPDATED event", "user_id", userID)
}

// ==================== READ OPERATIONS ====================

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

	// 2. Fetch from DB
	user, err := s.userRepo.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Set Cache (Async to not block)
	go func(u *models.User) {
		bytes, _ := json.Marshal(u)
		s.redisClient.Set(context.Background(), cacheKey, bytes, time.Hour*24)
	}(user)

	return user, nil
}

func (s *UserService) GetUsersByUsernames(ctx context.Context, usernames []string) ([]models.User, error) {
	if len(usernames) == 0 {
		return []models.User{}, nil
	}
	return s.userRepo.FindUsersByUsernames(ctx, usernames)
}

func (s *UserService) GetUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}

	// Deduplicate IDs
	uniqueIDs := make([]primitive.ObjectID, 0, len(ids))
	seen := make(map[string]struct{})
	for _, id := range ids {
		if _, ok := seen[id.Hex()]; !ok {
			uniqueIDs = append(uniqueIDs, id)
			seen[id.Hex()] = struct{}{}
		}
	}

	// 1. MGET from Cache
	keys := make([]string, len(uniqueIDs))
	orderMap := make(map[string]int)
	for i, id := range uniqueIDs {
		keys[i] = fmt.Sprintf("user:profile:%s", id.Hex())
		orderMap[id.Hex()] = i
	}

	users := make([]models.User, 0, len(uniqueIDs))
	missIDs := make([]primitive.ObjectID, 0)

	vals, err := s.redisClient.MGet(ctx, keys...).Result()
	if err == nil {
		for i, val := range vals {
			if val == nil {
				missIDs = append(missIDs, uniqueIDs[i])
				continue
			}
			var user models.User
			if strVal, ok := val.(string); ok {
				if err := json.Unmarshal([]byte(strVal), &user); err == nil {
					users = append(users, user)
					continue
				}
			}
			missIDs = append(missIDs, uniqueIDs[i])
		}
	} else {
		missIDs = uniqueIDs // All miss on Redis error
	}

	// 2. Fetch Misses from DB
	if len(missIDs) > 0 {
		dbUsers, err := s.userRepo.FindUsersByIDs(ctx, missIDs)
		if err != nil {
			return nil, err
		}

		// 3. Pipeline Set Cache for Misses
		if len(dbUsers) > 0 {
			go func(usersToCache []models.User) {
				pipe := s.redisClient.Pipeline()
				ctx := context.Background()
				for _, u := range usersToCache {
					bytes, _ := json.Marshal(u)
					pipe.Set(ctx, fmt.Sprintf("user:profile:%s", u.ID.Hex()), bytes, time.Hour*24)
				}
				if _, err := pipe.Exec(ctx); err != nil {
					s.logger.Error("Failed to pipeline cache set", "error", err)
				}
			}(dbUsers)
			users = append(users, dbUsers...)
		}
	}

	// Re-order to request order? Not strictly required by interface but good practice.
	// Current impl just appends.
	// If caller relies on order matching request 'ids', we should sort.
	// But standard FindUsersByIDs usually returns arbitrary order.

	return users, nil
}

// CheckRelationship determines the relationship status between two users (friend, blocked)
func (s *UserService) CheckRelationship(ctx context.Context, userID, targetID primitive.ObjectID) (isFriend, blockedByUser, blockedByTarget bool, err error) {
	if userID == targetID {
		return true, false, false, nil
	}

	// Batch fetch both users
	users, err := s.userRepo.FindUsersByIDs(ctx, []primitive.ObjectID{userID, targetID})
	if err != nil {
		return false, false, false, err
	}

	var user, target *models.User
	for i := range users {
		if users[i].ID == userID {
			user = &users[i]
		} else if users[i].ID == targetID {
			target = &users[i]
		}
	}

	if user == nil || target == nil {
		return false, false, false, errors.New("user not found")
	}

	// Check Friendship
	for _, friendID := range user.Friends {
		if friendID == targetID {
			isFriend = true
			break
		}
	}

	// Check Blocked By User
	for _, blockedID := range user.Blocked {
		if blockedID == targetID {
			blockedByUser = true
			break
		}
	}

	// Check Blocked By Target
	for _, blockedID := range target.Blocked {
		if blockedID == userID {
			blockedByTarget = true
			break
		}
	}

	return
}

func (s *UserService) ListUsers(ctx context.Context, page, limit int64, search string) ([]models.User, int64, error) {
	filter := bson.M{}
	if search != "" {
		filter["$or"] = []bson.M{
			{"username": bson.M{"$regex": search, "$options": "i"}},
			{"email": bson.M{"$regex": search, "$options": "i"}},
			{"full_name": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	total, err := s.userRepo.CountUsers(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip((page - 1) * limit).
		SetLimit(limit).
		SetSort(bson.D{{Key: "username", Value: 1}})

	users, err := s.userRepo.FindUsers(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	// Clear passwords
	for i := range users {
		users[i].Password = ""
	}

	return users, total, nil
}

func (s *UserService) GetUserStatus(ctx context.Context, userID string) (string, int64, error) {
	key := fmt.Sprintf("presence:%s", userID)
	val, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "offline", time.Now().Unix(), nil
		}
		return "", 0, err
	}

	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(val), &statusData); err != nil {
		return "offline", time.Now().Unix(), nil
	}

	status, _ := statusData["status"].(string)
	lastSeen, _ := statusData["last_seen"].(float64)
	return status, int64(lastSeen), nil
}

func (s *UserService) GetUsersPresence(ctx context.Context, userIDs []string) (map[string]map[string]interface{}, error) {
	presenceMap := make(map[string]map[string]interface{})

	keys := make([]string, len(userIDs))
	for i, userID := range userIDs {
		keys[i] = fmt.Sprintf("presence:%s", userID)
	}

	vals, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get presence from Redis: %w", err)
	}

	for i, userID := range userIDs {
		var statusData map[string]interface{}
		val := vals[i]

		if val == nil {
			statusData = map[string]interface{}{"status": "offline", "last_seen": time.Now().Unix()}
		} else {
			if err := json.Unmarshal([]byte(val.(string)), &statusData); err != nil {
				statusData = map[string]interface{}{"status": "offline", "last_seen": time.Now().Unix()}
			}
		}
		presenceMap[userID] = statusData
	}

	return presenceMap, nil
}

// ==================== WRITE OPERATIONS ====================

func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
	update["updated_at"] = time.Now()

	updatedUser, err := s.userRepo.UpdateUser(ctx, id, update)
	if err != nil {
		return nil, err
	}

	// Invalidate Cache
	if err := s.redisClient.Del(ctx, fmt.Sprintf("user:profile:%s", id.Hex())).Err(); err != nil {
		s.logger.Error("Failed to invalidate user cache", "user_id", id.Hex(), "error", err)
	}

	// Publish event for cache invalidation (legacy/downstream)
	s.publishUserUpdatedEvent(ctx, id.Hex(), updatedUser)
	if s.metrics != nil {
		s.metrics.IncrementProfileUpdates()
	}

	return updatedUser, nil
}

// UpdateProfileFields updates user profile fields
func (s *UserService) UpdateProfileFields(ctx context.Context, userID primitive.ObjectID, fullName, bio, avatar, coverPhoto, location, website string) (*models.User, error) {
	update := bson.M{}

	if fullName != "" {
		update["full_name"] = fullName
	}
	if bio != "" {
		update["bio"] = bio
	}
	if avatar != "" {
		update["avatar"] = avatar
	}
	if coverPhoto != "" {
		update["cover_photo"] = coverPhoto
	}
	if location != "" {
		update["location"] = location
	}
	if website != "" {
		update["website"] = website
	}

	if len(update) == 0 {
		return s.userRepo.FindUserByID(ctx, userID)
	}

	return s.UpdateUser(ctx, userID, update)
}

func (s *UserService) UpdateEmail(ctx context.Context, userID primitive.ObjectID, newEmail string) error {
	update := bson.M{
		"email":          newEmail,
		"email_verified": false,
		"updated_at":     time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		// Handle duplicate key error from unique index (atomic check)
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("email already in use by another account")
		}
		s.logger.Error("Failed to update email", "user_id", userID.Hex(), "error", err)
		return fmt.Errorf("failed to update email: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	if s.metrics != nil {
		s.metrics.IncrementEmailChanges()
	}
	s.logger.Info("Email updated successfully", "user_id", userID.Hex())
	return nil
}

func (s *UserService) UpdatePassword(ctx context.Context, userID primitive.ObjectID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	update := bson.M{
		"password":   string(hashedPassword),
		"updated_at": time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	if s.metrics != nil {
		s.metrics.IncrementPasswordChanges()
	}
	return nil
}

func (s *UserService) ToggleTwoFactor(ctx context.Context, userID primitive.ObjectID, enable bool) error {
	update := bson.M{
		"two_factor_enabled": enable,
		"updated_at":         time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to toggle two-factor authentication: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	return nil
}

func (s *UserService) DeactivateAccount(ctx context.Context, userID primitive.ObjectID) error {
	update := bson.M{
		"is_active":  false,
		"updated_at": time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate account: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	return nil
}

func (s *UserService) UpdatePublicKey(ctx context.Context, userID primitive.ObjectID, publicKey, encryptedPrivateKey, iv, salt string) error {
	update := bson.M{
		"public_key":            publicKey,
		"encrypted_private_key": encryptedPrivateKey,
		"key_backup_iv":         iv,
		"key_backup_salt":       salt,
		"updated_at":            time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to update keys: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	return nil
}

func (s *UserService) UpdatePrivacySettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdatePrivacySettingsRequest) error {
	updateFields := bson.M{
		"privacy_settings.last_updated": time.Now(),
	}

	if settings.DefaultPostPrivacy != "" {
		updateFields["privacy_settings.default_post_privacy"] = settings.DefaultPostPrivacy
	}
	if settings.CanSeeMyFriendsList != "" {
		updateFields["privacy_settings.can_see_my_friends_list"] = settings.CanSeeMyFriendsList
	}
	if settings.CanSendMeFriendRequests != "" {
		updateFields["privacy_settings.can_send_me_friend_requests"] = settings.CanSendMeFriendRequests
	}
	if settings.CanTagMeInPosts != "" {
		updateFields["privacy_settings.can_tag_me_in_posts"] = settings.CanTagMeInPosts
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, updateFields)
	if err != nil {
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	return nil
}

func (s *UserService) UpdateNotificationSettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdateNotificationSettingsRequest) error {
	updateFields := bson.M{}

	if settings.EmailNotifications != nil {
		updateFields["notification_settings.email_notifications"] = *settings.EmailNotifications
	}
	if settings.PushNotifications != nil {
		updateFields["notification_settings.push_notifications"] = *settings.PushNotifications
	}
	if settings.NotifyOnFriendRequest != nil {
		updateFields["notification_settings.notify_on_friend_request"] = *settings.NotifyOnFriendRequest
	}
	if settings.NotifyOnComment != nil {
		updateFields["notification_settings.notify_on_comment"] = *settings.NotifyOnComment
	}
	if settings.NotifyOnLike != nil {
		updateFields["notification_settings.notify_on_like"] = *settings.NotifyOnLike
	}
	if settings.NotifyOnTag != nil {
		updateFields["notification_settings.notify_on_tag"] = *settings.NotifyOnTag
	}
	if settings.NotifyOnMessage != nil {
		updateFields["notification_settings.notify_on_message"] = *settings.NotifyOnMessage
	}

	if len(updateFields) == 0 {
		return nil
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, updateFields)
	if err != nil {
		return fmt.Errorf("failed to update notification settings: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
	return nil
}
