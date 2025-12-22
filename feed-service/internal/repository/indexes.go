package repository

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// 3. Comments Indexes
	_, err = r.commentsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "post_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on comments(post_id): %v", err)
	}

	// 4. Replies Indexes
	_, err = r.repliesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "comment_id", Value: 1},
			{Key: "created_at", Value: 1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on replies(comment_id): %v", err)
	}

	// 5. Reactions Indexes
	// Unique Compound Index to prevent multiple reactions from same user on same target
	_, err = r.reactionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "target_id", Value: 1},
			{Key: "target_type", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Failed to create unique index on reactions: %v", err)
	}
	// Index for counting reactions by target
	_, err = r.reactionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "target_id", Value: 1},
			{Key: "target_type", Value: 1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on reactions(target): %v", err)
	}

	// 6. Albums Indexes
	_, err = r.albumsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on albums(user_id): %v", err)
	}

	// 7. AlbumMedia Indexes
	_, err = r.albumMediaCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "album_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Printf("Failed to create index on album_media(album_id): %v", err)
	}

	return nil
}
