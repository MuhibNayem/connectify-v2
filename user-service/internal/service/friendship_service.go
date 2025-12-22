package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"user-service/config"
	"user-service/internal/events"
	"user-service/internal/repository"

	"gitlab.com/spydotech-group/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FriendshipService struct {
	friendshipRepo *repository.FriendshipRepository
	graphRepo      *repository.GraphRepository
	userRepo       *repository.UserRepository
	producer       *events.EventProducer
	cfg            *config.Config
}

func NewFriendshipService(
	friendRepo *repository.FriendshipRepository,
	graphRepo *repository.GraphRepository,
	userRepo *repository.UserRepository,
	producer *events.EventProducer,
	cfg *config.Config,
) *FriendshipService {
	return &FriendshipService{
		friendshipRepo: friendRepo,
		graphRepo:      graphRepo,
		userRepo:       userRepo,
		producer:       producer,
		cfg:            cfg,
	}
}

func (s *FriendshipService) SendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	// 1. Create in Mongo (Legacy/Inbox)
	req, err := s.friendshipRepo.CreateRequest(ctx, requesterID, receiverID)
	if err != nil {
		return nil, err
	}

	// 2. Sync to Neo4j
	go s.graphRepo.SendRequest(context.Background(), requesterID, receiverID)

	// 3. Emit Event
	s.publishEvent("FriendRequestSent", requesterID, receiverID)

	return req, nil
}

func (s *FriendshipService) AcceptRequest(ctx context.Context, friendshipID, receiverID primitive.ObjectID) error {
	// 1. Authenticate Request ownership
	req, err := s.friendshipRepo.GetPendingFriendshipByID(ctx, friendshipID, receiverID)
	if err != nil || req == nil {
		return errors.New("request not found")
	}

	// 2. Update Mongo Status
	if err := s.friendshipRepo.UpdateStatus(ctx, friendshipID, receiverID, models.FriendshipStatusAccepted); err != nil {
		return err
	}

	// 3. Sync to Neo4j (Critical)
	go s.graphRepo.AcceptRequest(context.Background(), req.RequesterID, req.ReceiverID)

	// 4. Update Legacy Mongo Friends Array (Optional, but good for read compatibility)
	go s.userRepo.AddFriend(context.Background(), req.RequesterID, req.ReceiverID)

	// 5. Emit Event
	s.publishEvent("FriendRequestAccepted", req.RequesterID, req.ReceiverID)

	return nil
}

func (s *FriendshipService) RejectRequest(ctx context.Context, friendshipID, receiverID primitive.ObjectID) error {
	req, err := s.friendshipRepo.GetPendingFriendshipByID(ctx, friendshipID, receiverID)
	if err != nil || req == nil {
		return errors.New("request not found")
	}

	if err := s.friendshipRepo.UpdateStatus(ctx, friendshipID, receiverID, models.FriendshipStatusRejected); err != nil {
		return err
	}

	go s.graphRepo.RejectRequest(context.Background(), req.RequesterID, req.ReceiverID)

	return nil
}

func (s *FriendshipService) publishEvent(eventType string, actorID, targetID primitive.ObjectID) {
	event := map[string]interface{}{
		"event_type": eventType,
		"actor_id":   actorID.Hex(),
		"target_id":  targetID.Hex(),
		"timestamp":  time.Now(),
	}
	payload, _ := json.Marshal(event)
	go s.producer.Produce(context.Background(), []byte(actorID.Hex()), payload)
}
