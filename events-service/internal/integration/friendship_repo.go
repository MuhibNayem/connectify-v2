package integration

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FriendshipLocalRepository provides local access to replicated friendship data.
type FriendshipLocalRepository struct {
	collection *mongo.Collection
}

func NewFriendshipLocalRepository(db *mongo.Database) *FriendshipLocalRepository {
	return &FriendshipLocalRepository{
		collection: db.Collection("replicated_friendships"),
	}
}

func (r *FriendshipLocalRepository) GetFriends(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"requester_id": userID},
			{"receiver_id": userID},
		},
		"status": "accepted",
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friendIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var friendship EventFriendship
		if err := cursor.Decode(&friendship); err != nil {
			continue
		}

		if friendship.RequesterID == userID {
			friendIDs = append(friendIDs, friendship.ReceiverID)
		} else {
			friendIDs = append(friendIDs, friendship.RequesterID)
		}
	}
	return friendIDs, nil
}

// UpsertFriendship updates or inserts a replicated friendship
func (r *FriendshipLocalRepository) UpsertFriendship(ctx context.Context, friendship *EventFriendship) error {
	filter := bson.M{
		"requester_id": friendship.RequesterID,
		"receiver_id":  friendship.ReceiverID,
	}
	update := bson.M{
		"$set": friendship,
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// RemoveFriendship deletes a friendship
func (r *FriendshipLocalRepository) RemoveFriendship(ctx context.Context, requesterID, receiverID primitive.ObjectID) error {
	// Friendship order in graph/logic might vary, but in DB usually Requester/Receiver are fixed per request.
	// But to be safe, we delete matching pair.
	filter := bson.M{
		"$or": []bson.M{
			{"requester_id": requesterID, "receiver_id": receiverID},
			{"requester_id": receiverID, "receiver_id": requesterID},
		},
	}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}
