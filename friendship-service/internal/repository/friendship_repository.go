package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrCannotFriendSelf      = errors.New("cannot send friend request to yourself")
	ErrFriendRequestExists   = errors.New("friend request already exists between these users")
	ErrFriendRequestNotFound = errors.New("friend request not found or not actionable")
	ErrCannotBlockSelf       = errors.New("cannot block yourself")
	ErrAlreadyBlocked        = errors.New("user is already blocked")
	ErrFriendshipNotFound    = errors.New("friendship not found")
	ErrBlockNotFound         = errors.New("block relationship not found")
	ErrNotFriends            = errors.New("users are not friends")
)

type FriendshipRepository struct {
	db *mongo.Database
}

func NewFriendshipRepository(db *mongo.Database) *FriendshipRepository {
	// Ensure optimized indexes exist for MAANG-scale performance
	indexes := []mongo.IndexModel{
		// Primary unique index for bidirectional lookup
		{
			Keys: bson.D{
				{Key: "requester_id", Value: 1},
				{Key: "receiver_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		// Optimized: Status-based queries for requester
		{
			Keys: bson.D{
				{Key: "requester_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		// Optimized: Status-based queries for receiver
		{
			Keys: bson.D{
				{Key: "receiver_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		// Optimized: Partial index for pending requests only (smaller, faster)
		{
			Keys: bson.D{
				{Key: "receiver_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{"status": "pending"}),
		},
		// Text search index
		{
			Keys: bson.D{{Key: "$**", Value: "text"}},
		},
	}

	_, err := db.Collection("friendships").Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		log.Printf("Failed to create friendship indexes: %v", err)
	}

	return &FriendshipRepository{db: db}
}

func (r *FriendshipRepository) CreateRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	if requesterID == receiverID {
		return nil, ErrCannotFriendSelf
	}

	var existingFriendship models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{
		"$or": []bson.M{
			{
				"requester_id": requesterID,
				"receiver_id":  receiverID,
			},
			{
				"requester_id": receiverID,
				"receiver_id":  requesterID,
			},
		},
	}).Decode(&existingFriendship)

	if err == nil {
		if existingFriendship.Status == models.FriendshipStatusRejected {
			_, err := r.db.Collection("friendships").UpdateOne(ctx,
				bson.M{"_id": existingFriendship.ID},
				bson.M{
					"$set": bson.M{
						"requester_id": requesterID,
						"receiver_id":  receiverID,
						"status":       models.FriendshipStatusPending,
						"updated_at":   time.Now(),
					},
				},
			)
			if err != nil {
				return nil, err
			}
			existingFriendship.Status = models.FriendshipStatusPending
			existingFriendship.RequesterID = requesterID
			existingFriendship.ReceiverID = receiverID
			return &existingFriendship, nil
		}
		return nil, ErrFriendRequestExists
	} else if err != mongo.ErrNoDocuments {
		return nil, err
	}

	friendship := &models.Friendship{
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := r.db.Collection("friendships").InsertOne(ctx, friendship)
	if err != nil {
		return nil, err
	}

	friendship.ID = result.InsertedID.(primitive.ObjectID)
	return friendship, nil
}

func (r *FriendshipRepository) UpdateStatus(ctx context.Context, friendshipID primitive.ObjectID, receiverID primitive.ObjectID, status models.FriendshipStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	result, err := r.db.Collection("friendships").UpdateOne(
		ctx,
		bson.M{
			"_id":         friendshipID,
			"receiver_id": receiverID,
			"status":      models.FriendshipStatusPending,
		},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrFriendRequestNotFound
	}
	return nil
}

func (r *FriendshipRepository) AreFriends(ctx context.Context, userID1, userID2 primitive.ObjectID) (bool, error) {
	count, err := r.db.Collection("friendships").CountDocuments(ctx, bson.M{
		"status": models.FriendshipStatusAccepted,
		"$or": []bson.M{
			{
				"requester_id": userID1,
				"receiver_id":  userID2,
			},
			{
				"requester_id": userID2,
				"receiver_id":  userID1,
			},
		},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *FriendshipRepository) GetPendingRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	var friendship models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{
		"requester_id": requesterID,
		"receiver_id":  receiverID,
		"status":       models.FriendshipStatusPending,
	}).Decode(&friendship)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrFriendRequestNotFound
		}
		return nil, fmt.Errorf("failed to find pending request: %w", err)
	}
	return &friendship, nil
}

func (r *FriendshipRepository) GetPendingFriendshipByID(ctx context.Context, friendshipID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	var friendship models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{
		"_id":         friendshipID,
		"receiver_id": receiverID,
		"status":      models.FriendshipStatusPending,
	}).Decode(&friendship)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrFriendRequestNotFound
		}
		return nil, err
	}
	return &friendship, nil
}

func (r *FriendshipRepository) GetFriendRequests(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error) {
	matchFilter := bson.M{
		"status": status,
		"$or": []bson.M{
			{"requester_id": userID},
			{"receiver_id": userID},
		},
	}
	matchStage := bson.D{{Key: "$match", Value: matchFilter}}

	total, err := r.db.Collection("friendships").CountDocuments(ctx, matchFilter)
	if err != nil {
		return nil, 0, err
	}

	pipeline := mongo.Pipeline{
		matchStage,
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "requester_id",
			"foreignField": "_id",
			"as":           "requester_info",
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "receiver_id",
			"foreignField": "_id",
			"as":           "receiver_info",
		}}},
		bson.D{{Key: "$unwind", Value: "$requester_info"}},
		bson.D{{Key: "$unwind", Value: "$receiver_info"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":            1,
			"status":         1,
			"created_at":     1,
			"updated_at":     1,
			"requester_id":   1,
			"receiver_id":    1,
			"requester_info": "$requester_info",
			"receiver_info":  "$receiver_info",
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "updated_at", Value: -1}}}},
		bson.D{{Key: "$skip", Value: (page - 1) * limit}},
		bson.D{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.db.Collection("friendships").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	requests := make([]models.PopulatedFriendship, 0)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

func (r *FriendshipRepository) Unfriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	areFriends, err := r.AreFriends(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !areFriends {
		return ErrNotFriends
	}

	result, err := r.db.Collection("friendships").DeleteOne(ctx, bson.M{
		"status": models.FriendshipStatusAccepted,
		"$or": []bson.M{
			{
				"requester_id": userID,
				"receiver_id":  friendID,
			},
			{
				"requester_id": friendID,
				"receiver_id":  userID,
			},
		},
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrFriendshipNotFound
	}
	return nil
}

func (r *FriendshipRepository) BlockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}
	alreadyBlocked, err := r.IsBlocked(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if alreadyBlocked {
		return ErrAlreadyBlocked
	}
	areFriends, err := r.AreFriends(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if areFriends {
		// Clean up existing friendship
		_, _ = r.db.Collection("friendships").DeleteMany(ctx, bson.M{
			"$or": []bson.M{
				{
					"requester_id": blockerID,
					"receiver_id":  blockedID,
				},
				{
					"requester_id": blockedID,
					"receiver_id":  blockerID,
				},
			},
		})
	}

	blockedFriendship := &models.Friendship{
		RequesterID: blockerID,
		ReceiverID:  blockedID,
		Status:      models.FriendshipStatusBlocked,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err = r.db.Collection("friendships").InsertOne(ctx, blockedFriendship)
	return err
}

func (r *FriendshipRepository) UnblockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	isBlocked, err := r.IsBlockedBy(ctx, blockedID, blockerID)
	if err != nil {
		return err
	}
	if !isBlocked {
		return ErrBlockNotFound
	}
	result, err := r.db.Collection("friendships").DeleteOne(ctx, bson.M{
		"requester_id": blockerID,
		"receiver_id":  blockedID,
		"status":       models.FriendshipStatusBlocked,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrBlockNotFound
	}
	return nil
}

func (r *FriendshipRepository) IsBlockedBy(ctx context.Context, blockedID, blockerID primitive.ObjectID) (bool, error) {
	count, err := r.db.Collection("friendships").CountDocuments(ctx, bson.M{
		"requester_id": blockerID,
		"receiver_id":  blockedID,
		"status":       models.FriendshipStatusBlocked,
	})
	return count > 0, err
}

func (r *FriendshipRepository) IsBlocked(ctx context.Context, userID1, userID2 primitive.ObjectID) (bool, error) {
	count, err := r.db.Collection("friendships").CountDocuments(ctx, bson.M{
		"status": models.FriendshipStatusBlocked,
		"$or": []bson.M{
			{
				"requester_id": userID1,
				"receiver_id":  userID2,
			},
			{
				"requester_id": userID2,
				"receiver_id":  userID1,
			},
		},
	})
	return count > 0, err
}

func (r *FriendshipRepository) GetBlockedUsers(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	cursor, err := r.db.Collection("friendships").Find(ctx, bson.M{
		"requester_id": userID,
		"status":       models.FriendshipStatusBlocked,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blockedUsers []primitive.ObjectID
	for cursor.Next(ctx) {
		var friendship models.Friendship
		if err := cursor.Decode(&friendship); err != nil {
			return nil, err
		}
		blockedUsers = append(blockedUsers, friendship.ReceiverID)
	}
	return blockedUsers, nil
}

func (r *FriendshipRepository) SearchFriends(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error) {
	regexPattern := fmt.Sprintf(".*%s.*", query)
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"status": models.FriendshipStatusAccepted,
			"$or": []bson.M{
				{"requester_id": userID},
				{"receiver_id": userID},
			},
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "requester_id",
			"foreignField": "_id",
			"as":           "requester_info",
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "receiver_id",
			"foreignField": "_id",
			"as":           "receiver_info",
		}}},
		bson.D{{Key: "$unwind", Value: "$requester_info"}},
		bson.D{{Key: "$unwind", Value: "$receiver_info"}},
		bson.D{{Key: "$project", Value: bson.M{
			"friend_info": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$requester_id", userID}},
					"$receiver_info",
					"$requester_info",
				},
			},
		}}},
		bson.D{{Key: "$match", Value: bson.M{
			"$or": []bson.M{
				{"friend_info.username": bson.M{"$regex": regexPattern, "$options": "i"}},
				{"friend_info.full_name": bson.M{"$regex": regexPattern, "$options": "i"}},
			},
		}}},
		bson.D{{Key: "$limit", Value: limit}},
		bson.D{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$friend_info"}}},
	}

	cursor, err := r.db.Collection("friendships").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friends []models.UserShortResponse
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, err
	}
	return friends, nil
}

// ===== Transaction-compatible methods =====

var ErrConcurrentModification = errors.New("concurrent modification detected")

// CreateRequestTx creates a friend request within a transaction
func (r *FriendshipRepository) CreateRequestTx(ctx mongo.SessionContext, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	return r.CreateRequest(ctx, requesterID, receiverID)
}

// UpdateStatusWithVersion uses optimistic locking for concurrent update protection
func (r *FriendshipRepository) UpdateStatusWithVersion(ctx context.Context, friendshipID primitive.ObjectID, expectedVersion int64, status models.FriendshipStatus) error {
	result, err := r.db.Collection("friendships").UpdateOne(
		ctx,
		bson.M{
			"_id":     friendshipID,
			"version": expectedVersion,
		},
		bson.M{
			"$set": bson.M{
				"status":     status,
				"updated_at": time.Now(),
			},
			"$inc": bson.M{
				"version": 1,
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrConcurrentModification
	}
	return nil
}

// GetAllAccepted retrieves all accepted friendships for reconciliation
func (r *FriendshipRepository) GetAllAccepted(ctx context.Context, limit int64) ([]models.Friendship, error) {
	filter := bson.M{"status": models.FriendshipStatusAccepted}

	opts := options.Find().SetLimit(limit)

	cursor, err := r.db.Collection("friendships").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friendships []models.Friendship
	if err := cursor.All(ctx, &friendships); err != nil {
		return nil, err
	}
	return friendships, nil
}

// GetByID retrieves a friendship by ID
func (r *FriendshipRepository) GetByID(ctx context.Context, friendshipID primitive.ObjectID) (*models.Friendship, error) {
	var friendship models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{"_id": friendshipID}).Decode(&friendship)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrFriendshipNotFound
		}
		return nil, err
	}
	return &friendship, nil
}

// GetDatabase returns the database for transaction support
func (r *FriendshipRepository) GetDatabase() *mongo.Database {
	return r.db
}
