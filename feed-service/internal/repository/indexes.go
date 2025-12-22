package repository

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *FeedRepository) EnsureIndexes(ctx context.Context) error {
	// 1. Posts Indexes
	// Compound index for Feed Query: (user_id, privacy, status, created_at) is hard because of OR condition.
	// We need indexes to support the $or clauses.

	// Index for "My Posts"
	_, err := r.postsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "status", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on posts(user_id): %v", err)
	}

	// Index for "Public Posts"
	// Ideally partial index where privacy=PUBLIC
	_, err = r.postsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "privacy", Value: 1},
			{Key: "status", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on posts(privacy): %v", err)
	}

	// 2. Friendships Indexes
	// We query by requester_id OR receiver_id.
	_, err = r.friendshipsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "requester_id", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "receiver_id", Value: 1}, {Key: "status", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
