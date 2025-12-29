package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/cache"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/kafka"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/validation"
	"github.com/MuhibNayem/connectify-v2/shared-entity/events"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	kafkalib "github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FriendshipService handles all friendship-related business logic
type FriendshipService struct {
	friendshipRepo *repository.FriendshipRepository
	userClient     *repository.UserClient
	graphRepo      *repository.GraphRepository
	kafkaProducer  *kafka.MessageProducer
	cache          *cache.FriendshipCache
	breaker        *CircuitBreakerWrapper
	metrics        *metrics.BusinessMetrics
	validator      *validation.Validator
	logger         *slog.Logger
}

// NewFriendshipService creates a new FriendshipService with all dependencies
func NewFriendshipService(
	fr *repository.FriendshipRepository,
	uc *repository.UserClient,
	gr *repository.GraphRepository,
	kp *kafka.MessageProducer,
	c *cache.FriendshipCache,
	cb *CircuitBreakerWrapper,
	m *metrics.BusinessMetrics,
	v *validation.Validator,
	l *slog.Logger,
) *FriendshipService {
	if l == nil {
		l = slog.Default()
	}
	return &FriendshipService{
		friendshipRepo: fr,
		userClient:     uc,
		graphRepo:      gr,
		kafkaProducer:  kp,
		cache:          c,
		breaker:        cb,
		metrics:        m,
		validator:      v,
		logger:         l,
	}
}

func (s *FriendshipService) publishEvent(ctx context.Context, requesterID, receiverID string, status, action string) {
	if s.kafkaProducer == nil {
		return
	}

	event := events.FriendshipEvent{
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      status,
		Action:      action,
		Timestamp:   time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal event", "error", err)
		return
	}

	msg := kafkalib.Message{
		Key:   []byte(requesterID),
		Value: payload,
		Time:  time.Now(),
	}

	// Use circuit breaker for Kafka operations
	if s.breaker != nil {
		err = s.breaker.ExecuteKafkaOp(ctx, "publish_event", func() error {
			return s.kafkaProducer.ProduceMessage(ctx, msg)
		})
	} else {
		err = s.kafkaProducer.ProduceMessage(ctx, msg)
	}

	if err != nil {
		s.logger.Error("Failed to publish event", "error", err, "action", action)
		if s.metrics != nil {
			s.metrics.RecordOperationError("publish_event")
		}
	}
}

// SendRequest sends a friend request
func (s *FriendshipService) SendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveOperationDuration("send_request", time.Since(start).Seconds())
		}
	}()

	// Validate
	if s.validator != nil {
		if err := s.validator.ValidateFriendRequest(requesterID, receiverID); err != nil {
			return nil, err
		}
	}

	// Sync graph user nodes with circuit breaker
	if s.graphRepo != nil && s.breaker != nil {
		_ = s.breaker.ExecuteGraphOp(ctx, "sync_user", func() error {
			_ = s.graphRepo.SyncUser(ctx, requesterID)
			return s.graphRepo.SyncUser(ctx, receiverID)
		})
	} else if s.graphRepo != nil {
		_ = s.graphRepo.SyncUser(ctx, requesterID)
		_ = s.graphRepo.SyncUser(ctx, receiverID)
	}

	// Create MongoDB request
	friendship, err := s.friendshipRepo.CreateRequest(ctx, requesterID, receiverID)
	if err != nil {
		s.logger.Error("Failed to create friend request", "error", err)
		if s.metrics != nil {
			s.metrics.RecordOperationError("send_request")
		}
		return nil, err
	}

	// Create Graph request with circuit breaker
	if s.graphRepo != nil {
		graphErr := func() error {
			if s.breaker != nil {
				return s.breaker.ExecuteGraphOp(ctx, "send_request", func() error {
					return s.graphRepo.SendRequest(ctx, requesterID, receiverID)
				})
			}
			return s.graphRepo.SendRequest(ctx, requesterID, receiverID)
		}()
		if graphErr != nil {
			s.logger.Warn("Graph SendRequest error", "error", graphErr)
		}
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateFriendship(ctx, requesterID, receiverID)
	}

	// Record metric and publish event
	if s.metrics != nil {
		s.metrics.RecordRequestSent()
	}
	go s.publishEvent(context.Background(), requesterID.Hex(), receiverID.Hex(), "pending", "request")

	return friendship, nil
}

// RespondToRequest accepts or rejects a friend request
func (s *FriendshipService) RespondToRequest(ctx context.Context, friendshipID, receiverID primitive.ObjectID, accept bool) error {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveOperationDuration("respond_to_request", time.Since(start).Seconds())
		}
	}()

	targetRequest, err := s.friendshipRepo.GetPendingFriendshipByID(ctx, friendshipID, receiverID)
	if err != nil {
		return err
	}

	status := models.FriendshipStatusRejected
	if accept {
		status = models.FriendshipStatusAccepted

		// Update Users collection via User Service with circuit breaker
		addFriendErr := func() error {
			if s.breaker != nil {
				return s.breaker.ExecuteUserServiceOp(ctx, "add_friend", func() error {
					if err := s.userClient.AddFriend(ctx, targetRequest.RequesterID, targetRequest.ReceiverID); err != nil {
						return err
					}
					return s.userClient.AddFriend(ctx, targetRequest.ReceiverID, targetRequest.RequesterID)
				})
			}
			if err := s.userClient.AddFriend(ctx, targetRequest.RequesterID, targetRequest.ReceiverID); err != nil {
				return err
			}
			return s.userClient.AddFriend(ctx, targetRequest.ReceiverID, targetRequest.RequesterID)
		}()
		if addFriendErr != nil {
			s.logger.Error("Failed to add friends via user service", "error", addFriendErr)
			if s.metrics != nil {
				s.metrics.RecordOperationError("respond_to_request")
			}
			return addFriendErr
		}
	}

	// Update Friendship status
	if err := s.friendshipRepo.UpdateStatus(ctx, friendshipID, receiverID, status); err != nil {
		return err
	}

	// Update Graph
	if s.graphRepo != nil {
		graphFn := func() error {
			if accept {
				return s.graphRepo.AcceptRequest(ctx, targetRequest.RequesterID, targetRequest.ReceiverID)
			}
			return s.graphRepo.RejectRequest(ctx, targetRequest.RequesterID, targetRequest.ReceiverID)
		}
		if s.breaker != nil {
			_ = s.breaker.ExecuteGraphOp(ctx, "respond_request", graphFn)
		} else {
			_ = graphFn()
		}
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateFriendship(ctx, targetRequest.RequesterID, targetRequest.ReceiverID)
	}

	// Record metrics and publish event
	if s.metrics != nil {
		if accept {
			s.metrics.RecordRequestAccepted()
		} else {
			s.metrics.RecordRequestRejected()
		}
	}

	action := "reject"
	statusStr := "rejected"
	if accept {
		action = "accept"
		statusStr = "accepted"
	}
	go s.publishEvent(context.Background(), targetRequest.RequesterID.Hex(), targetRequest.ReceiverID.Hex(), statusStr, action)

	return nil
}

// ListFriendships returns paginated friendships
func (s *FriendshipService) ListFriendships(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error) {
	if s.validator != nil {
		if err := s.validator.ValidatePagination(page, limit); err != nil {
			return nil, 0, err
		}
	}
	return s.friendshipRepo.GetFriendRequests(ctx, userID, status, page, limit)
}

// CheckFriendship checks if two users are friends
func (s *FriendshipService) CheckFriendship(ctx context.Context, userID1, userID2 primitive.ObjectID) (bool, error) {
	// Try cache first
	if s.cache != nil {
		cached, err := s.cache.GetFriendshipStatus(ctx, userID1, userID2)
		if err == nil && cached != nil && *cached == models.FriendshipStatusAccepted {
			return true, nil
		}
	}

	if s.graphRepo != nil {
		var areFriends bool
		var graphErr error
		if s.breaker != nil {
			graphErr = s.breaker.ExecuteGraphOp(ctx, "check_friendship", func() error {
				var err error
				areFriends, _, _, _, _, err = s.graphRepo.CheckFriendshipStatus(ctx, userID1, userID2)
				return err
			})
		} else {
			areFriends, _, _, _, _, graphErr = s.graphRepo.CheckFriendshipStatus(ctx, userID1, userID2)
		}
		if graphErr == nil {
			return areFriends, nil
		}
		s.logger.Warn("Graph check failed, falling back to Mongo", "error", graphErr)
	}

	return s.friendshipRepo.AreFriends(ctx, userID1, userID2)
}

// Unfriend removes a friendship
func (s *FriendshipService) Unfriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveOperationDuration("unfriend", time.Since(start).Seconds())
		}
	}()

	// Remove from User Service
	if s.breaker != nil {
		_ = s.breaker.ExecuteUserServiceOp(ctx, "remove_friend", func() error {
			_ = s.userClient.RemoveFriend(ctx, userID, friendID)
			return s.userClient.RemoveFriend(ctx, friendID, userID)
		})
	} else {
		_ = s.userClient.RemoveFriend(ctx, userID, friendID)
		_ = s.userClient.RemoveFriend(ctx, friendID, userID)
	}

	// Remove from MongoDB
	err := s.friendshipRepo.Unfriend(ctx, userID, friendID)
	if err != nil && !errors.Is(err, repository.ErrNotFriends) && !errors.Is(err, repository.ErrFriendshipNotFound) {
		s.logger.Error("Mongo Unfriend error", "error", err)
	}

	// Remove from Graph
	if s.graphRepo != nil {
		if s.breaker != nil {
			_ = s.breaker.ExecuteGraphOp(ctx, "unfriend", func() error {
				return s.graphRepo.Unfriend(ctx, userID, friendID)
			})
		} else {
			_ = s.graphRepo.Unfriend(ctx, userID, friendID)
		}
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateFriendship(ctx, userID, friendID)
	}

	if s.metrics != nil {
		s.metrics.RecordUnfriend()
	}
	go s.publishEvent(context.Background(), userID.Hex(), friendID.Hex(), "removed", "remove")
	return nil
}

// BlockUser blocks a user
func (s *FriendshipService) BlockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveOperationDuration("block_user", time.Since(start).Seconds())
		}
	}()

	if s.validator != nil {
		if err := s.validator.ValidateBlock(blockerID, blockedID); err != nil {
			return err
		}
	}

	if err := s.friendshipRepo.BlockUser(ctx, blockerID, blockedID); err != nil {
		if !errors.Is(err, repository.ErrAlreadyBlocked) {
			return err
		}
	}

	if s.graphRepo != nil {
		if s.breaker != nil {
			_ = s.breaker.ExecuteGraphOp(ctx, "block_user", func() error {
				return s.graphRepo.BlockUser(ctx, blockerID, blockedID)
			})
		} else {
			_ = s.graphRepo.BlockUser(ctx, blockerID, blockedID)
		}
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateBlockStatus(ctx, blockerID, blockedID)
		_ = s.cache.InvalidateFriendship(ctx, blockerID, blockedID)
	}

	if s.metrics != nil {
		s.metrics.RecordBlock()
	}
	go s.publishEvent(context.Background(), blockerID.Hex(), blockedID.Hex(), "blocked", "block")
	return nil
}

// UnblockUser unblocks a user
func (s *FriendshipService) UnblockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	_ = s.friendshipRepo.UnblockUser(ctx, blockerID, blockedID)

	if s.graphRepo != nil {
		if s.breaker != nil {
			_ = s.breaker.ExecuteGraphOp(ctx, "unblock_user", func() error {
				return s.graphRepo.UnblockUser(ctx, blockerID, blockedID)
			})
		} else {
			_ = s.graphRepo.UnblockUser(ctx, blockerID, blockedID)
		}
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateBlockStatus(ctx, blockerID, blockedID)
	}

	if s.metrics != nil {
		s.metrics.RecordUnblock()
	}
	go s.publishEvent(context.Background(), blockerID.Hex(), blockedID.Hex(), "unblocked", "unblock")
	return nil
}

// GetBlockedUsers returns users blocked by the given user
func (s *FriendshipService) GetBlockedUsers(ctx context.Context, userID primitive.ObjectID) ([]models.User, error) {
	blockedIDs, err := s.friendshipRepo.GetBlockedUsers(ctx, userID)
	if err != nil {
		return nil, err
	}

	var blockedUsers []models.User
	for _, id := range blockedIDs {
		blockedUsers = append(blockedUsers, models.User{ID: id})
	}
	return blockedUsers, nil
}

// FriendshipStatusResponse contains detailed friendship status
type FriendshipStatusResponse struct {
	AreFriends        bool
	RequestSent       bool
	RequestReceived   bool
	IsBlockedByViewer bool
	HasBlockedViewer  bool
}

// GetDetailedFriendshipStatus returns comprehensive status between two users
func (s *FriendshipService) GetDetailedFriendshipStatus(ctx context.Context, viewerID, otherUserID primitive.ObjectID) (*FriendshipStatusResponse, error) {
	status := &FriendshipStatusResponse{}

	areFriends, err := s.friendshipRepo.AreFriends(ctx, viewerID, otherUserID)
	if err != nil {
		return nil, err
	}
	status.AreFriends = areFriends

	reqSent, err := s.friendshipRepo.GetPendingRequest(ctx, viewerID, otherUserID)
	if err == nil && reqSent != nil {
		status.RequestSent = true
	}

	reqRecv, err := s.friendshipRepo.GetPendingRequest(ctx, otherUserID, viewerID)
	if err == nil && reqRecv != nil {
		status.RequestReceived = true
	}

	isBlocked, err := s.friendshipRepo.IsBlockedBy(ctx, viewerID, otherUserID)
	if err != nil {
		return nil, err
	}
	status.IsBlockedByViewer = isBlocked

	hasBlocked, err := s.friendshipRepo.IsBlockedBy(ctx, otherUserID, viewerID)
	if err != nil {
		return nil, err
	}
	status.HasBlockedViewer = hasBlocked

	return status, nil
}

// SearchFriends searches for friends matching a query
func (s *FriendshipService) SearchFriends(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error) {
	if s.validator != nil {
		if err := s.validator.ValidateSearch(userID, query, limit); err != nil {
			return nil, err
		}
	}
	return s.friendshipRepo.SearchFriends(ctx, userID, query, limit)
}
