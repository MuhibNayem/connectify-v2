package repository

import (
	"context"
	"errors"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FriendshipRepository struct {
	db *mongo.Database
}

func NewFriendshipRepository(db *mongo.Database) *FriendshipRepository {
	_, _ = db.Collection("friendships").Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "requester_id", Value: 1},
				{Key: "receiver_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(30 * 24 * 60 * 60)}, // 30 days
	})
	return &FriendshipRepository{db: db}
}

func (r *FriendshipRepository) CreateRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	friendship := &models.Friendship{
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	res, err := r.db.Collection("friendships").InsertOne(ctx, friendship)
	if err != nil {
		return nil, err // Simplify error handling for now
	}
	friendship.ID = res.InsertedID.(primitive.ObjectID)
	return friendship, nil
}

func (r *FriendshipRepository) UpdateStatus(ctx context.Context, friendshipID, receiverID primitive.ObjectID, status models.FriendshipStatus) error {
	_, err := r.db.Collection("friendships").UpdateOne(
		ctx,
		bson.M{"_id": friendshipID, "receiver_id": receiverID},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	return err
}

func (r *FriendshipRepository) GetPendingRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	var f models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{
		"requester_id": requesterID, "receiver_id": receiverID, "status": models.FriendshipStatusPending,
	}).Decode(&f)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &f, err
}

func (r *FriendshipRepository) GetPendingFriendshipByID(ctx context.Context, id, receiverID primitive.ObjectID) (*models.Friendship, error) {
	var f models.Friendship
	err := r.db.Collection("friendships").FindOne(ctx, bson.M{"_id": id, "receiver_id": receiverID, "status": models.FriendshipStatusPending}).Decode(&f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FriendshipRepository) Unfriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	_, err := r.db.Collection("friendships").DeleteOne(ctx, bson.M{
		"status": models.FriendshipStatusAccepted,
		"$or": []bson.M{
			{"requester_id": userID, "receiver_id": friendID},
			{"requester_id": friendID, "receiver_id": userID},
		},
	})
	return err
}

func (r *FriendshipRepository) GetFriendRequests(ctx context.Context, userID primitive.ObjectID) ([]models.Friendship, error) {
	cursor, err := r.db.Collection("friendships").Find(ctx, bson.M{
		"receiver_id": userID,
		"status":      models.FriendshipStatusPending,
	})
	if err != nil {
		return nil, err
	}
	var requests []models.Friendship
	_ = cursor.All(ctx, &requests)
	return requests, nil
}

// Errors
var ErrFriendRequestNotFound = errors.New("friend request not found")
