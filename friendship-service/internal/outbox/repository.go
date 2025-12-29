package outbox

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles outbox event persistence
type Repository struct {
	collection *mongo.Collection
}

// NewRepository creates a new outbox repository
func NewRepository(db *mongo.Database) *Repository {
	collection := db.Collection("outbox_events")

	// Create indexes for efficient querying
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "next_retry", Value: 1},
			},
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(7 * 24 * 60 * 60), // TTL: 7 days
		},
	}
	_, _ = collection.Indexes().CreateMany(context.Background(), indexes)

	return &Repository{collection: collection}
}

// Insert inserts a new outbox event
func (r *Repository) Insert(ctx context.Context, event *Event) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

// InsertTx inserts a new outbox event within a transaction
func (r *Repository) InsertTx(ctx mongo.SessionContext, event *Event) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

// GetPending retrieves pending events ready for processing
func (r *Repository) GetPending(ctx context.Context, limit int64) ([]*Event, error) {
	filter := bson.M{
		"status": StatusPending,
		"next_retry": bson.M{
			"$lte": time.Now(),
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// MarkProcessed marks an event as successfully processed
func (r *Repository) MarkProcessed(ctx context.Context, eventID primitive.ObjectID) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": eventID},
		bson.M{
			"$set": bson.M{
				"status":       StatusProcessed,
				"processed_at": now,
			},
		},
	)
	return err
}

// MarkFailed marks an event for retry with exponential backoff
func (r *Repository) MarkFailed(ctx context.Context, eventID primitive.ObjectID, errMsg string, retryCount int, maxRetries int) error {
	var status EventStatus
	var nextRetry time.Time

	if retryCount >= maxRetries {
		status = StatusDead
		nextRetry = time.Now().Add(100 * 365 * 24 * time.Hour) // Never retry
	} else {
		status = StatusPending
		// Exponential backoff: 2^retry * 1 second (1s, 2s, 4s, 8s, 16s)
		backoff := time.Duration(1<<uint(retryCount)) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute // Cap at 5 minutes
		}
		nextRetry = time.Now().Add(backoff)
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": eventID},
		bson.M{
			"$set": bson.M{
				"status":      status,
				"error":       errMsg,
				"retry_count": retryCount + 1,
				"next_retry":  nextRetry,
			},
		},
	)
	return err
}

// GetDeadLetterEvents retrieves events that exceeded max retries
func (r *Repository) GetDeadLetterEvents(ctx context.Context, limit int64) ([]*Event, error) {
	filter := bson.M{"status": StatusDead}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}
