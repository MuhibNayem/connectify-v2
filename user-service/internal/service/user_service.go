package service

import (
	"context"
	"encoding/json"
	"time"
	"user-service/config"
	"user-service/internal/events"
	"user-service/internal/repository"

	"gitlab.com/spydotech-group/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService struct {
	userRepo *repository.UserRepository
	producer *events.EventProducer
	cfg      *config.Config
}

func NewUserService(userRepo *repository.UserRepository, producer *events.EventProducer, cfg *config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		producer: producer,
		cfg:      cfg,
	}
}

func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	return s.userRepo.FindUserByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
	updatedUser, err := s.userRepo.UpdateUser(ctx, id, update)
	if err != nil {
		return nil, err
	}

	// Publish UserUpdated Event
	event := map[string]interface{}{
		"event_type":     "UserUpdated",
		"user_id":        id.Hex(),
		"updated_fields": update, // Send what changed
		"timestamp":      time.Now(),
		"user_data":      updatedUser, // Send full new state (optional but helpful)
	}

	payload, _ := json.Marshal(event)
	// Fire and forget event
	go s.producer.Produce(context.Background(), []byte(id.Hex()), payload)

	return updatedUser, nil
}
