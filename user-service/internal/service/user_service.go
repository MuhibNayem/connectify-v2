package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"user-service/config"
	"user-service/internal/events"
	"user-service/internal/repository"

	"github.com/redis/go-redis/v9"
	"gitlab.com/spydotech-group/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo    *repository.UserRepository
	producer    *events.EventProducer
	redisClient redis.UniversalClient
	cfg         *config.Config
}

func NewUserService(userRepo *repository.UserRepository, producer *events.EventProducer, redisClient redis.UniversalClient, cfg *config.Config) *UserService {
	return &UserService{
		userRepo:    userRepo,
		producer:    producer,
		redisClient: redisClient,
		cfg:         cfg,
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

	payload, _ := json.Marshal(event)
	go s.producer.Produce(context.Background(), []byte(userID), payload)
	log.Printf("Published USER_UPDATED event for user %s", userID)
}

// ==================== READ OPERATIONS ====================

func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	return s.userRepo.FindUserByID(ctx, id)
}

func (s *UserService) GetUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	users := make([]models.User, 0, len(ids))
	for _, id := range ids {
		user, err := s.userRepo.FindUserByID(ctx, id)
		if err == nil && user != nil {
			users = append(users, *user)
		}
	}
	return users, nil
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

	// Publish event for cache invalidation
	s.publishUserUpdatedEvent(ctx, id.Hex(), updatedUser)

	return updatedUser, nil
}

func (s *UserService) UpdateEmail(ctx context.Context, userID primitive.ObjectID, newEmail string) error {
	// Check if email already exists
	existingUser, err := s.userRepo.FindUserByEmail(ctx, newEmail)
	if err == nil && existingUser != nil && existingUser.ID != userID {
		return errors.New("email already in use by another account")
	}

	update := bson.M{
		"email":          newEmail,
		"email_verified": false,
		"updated_at":     time.Now(),
	}

	updatedUser, err := s.userRepo.UpdateUser(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	s.publishUserUpdatedEvent(ctx, userID.Hex(), updatedUser)
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
