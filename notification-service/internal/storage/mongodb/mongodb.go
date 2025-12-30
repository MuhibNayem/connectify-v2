package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoStorage(uri, database, collection string) (*MongoStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	coll := client.Database(database).Collection(collection)

	// Create indexes
	_, err = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "read", Value: 1}}},
		{Keys: bson.D{{Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoStorage{
		client:     client,
		collection: coll,
	}, nil
}

func (m *MongoStorage) Create(ctx context.Context, notification *adapters.Notification) error {
	if notification.ID == "" {
		notification.ID = primitive.NewObjectID().Hex()
	}

	now := time.Now()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)

	doc := m.toDocument(notification)
	_, err := m.collection.InsertOne(ctx, doc)
	return err
}

func (m *MongoStorage) Get(ctx context.Context, id string) (*adapters.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	var doc bson.M
	err = m.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		return nil, err
	}

	return m.fromDocument(doc), nil
}

func (m *MongoStorage) List(ctx context.Context, query *adapters.ListQuery) (*adapters.ListResult, error) {
	filter := bson.M{"recipient_id": query.RecipientID}

	if query.Read != nil {
		filter["read"] = *query.Read
	}
	if len(query.Types) > 0 {
		filter["type"] = bson.M{"$in": query.Types}
	}
	if query.Priority != "" {
		filter["priority"] = query.Priority
	}
	if query.Since != nil {
		filter["created_at"] = bson.M{"$gte": *query.Since}
	}
	if query.Until != nil {
		if filter["created_at"] != nil {
			filter["created_at"].(bson.M)["$lte"] = *query.Until
		} else {
			filter["created_at"] = bson.M{"$lte": *query.Until}
		}
	}

	// Count total
	total, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Query with pagination
	limit := int64(query.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sortOrder := -1
	if query.Sort == "created_at" {
		sortOrder = 1
	}

	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "created_at", Value: sortOrder}})

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*adapters.Notification
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		notifications = append(notifications, m.fromDocument(doc))
	}

	return &adapters.ListResult{
		Notifications: notifications,
		Total:         total,
		HasMore:       int64(len(notifications)) >= limit,
	}, nil
}

func (m *MongoStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	updates["updated_at"] = time.Now().Format(time.RFC3339)
	if read, ok := updates["read"].(bool); ok && read {
		updates["read_at"] = time.Now().Format(time.RFC3339)
	}

	_, err = m.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)
	return err
}

func (m *MongoStorage) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	_, err = m.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (m *MongoStorage) GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return m.collection.CountDocuments(ctx, bson.M{
		"recipient_id": recipientID,
		"read":         false,
	})
}

func (m *MongoStorage) BatchMarkAsRead(ctx context.Context, recipientID string, ids []string) (int64, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, objectID)
		}
	}

	result, err := m.collection.UpdateMany(
		ctx,
		bson.M{
			"_id":          bson.M{"$in": objectIDs},
			"recipient_id": recipientID,
		},
		bson.M{
			"$set": bson.M{
				"read":       true,
				"read_at":    time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *MongoStorage) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := m.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lt":     now,
		},
	})
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

func (m *MongoStorage) HealthCheck(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}

func (m *MongoStorage) SaveDeliveryState(ctx context.Context, state *models.NotificationDeliveryState) error {
	return nil // Stub
}

func (m *MongoStorage) GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error) {
	return nil, nil // Stub
}

func (m *MongoStorage) CreateWithOutbox(ctx context.Context, notification *adapters.Notification, event *adapters.NotificationEvent) error {
	// TODO: Implement multi-document transaction
	return m.Create(ctx, notification)
}

func (m *MongoStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
	return nil, nil // Stub
}

func (m *MongoStorage) DeleteOutboxEvent(ctx context.Context, eventID string) error {
	return nil // Stub
}

func (m *MongoStorage) Close() error {
	return m.client.Disconnect(context.Background())
}

func (m *MongoStorage) toDocument(n *adapters.Notification) bson.M {
	doc := bson.M{
		"recipient_id": n.RecipientID,
		"sender_id":    n.SenderID,
		"type":         n.Type,
		"title":        n.Title,
		"body":         n.Body,
		"priority":     n.Priority,
		"channels":     n.Channels,
		"read":         n.Read,
		"created_at":   n.CreatedAt,
		"updated_at":   n.UpdatedAt,
	}

	if id, err := primitive.ObjectIDFromHex(n.ID); err == nil {
		doc["_id"] = id
	}
	if n.ImageURL != "" {
		doc["image_url"] = n.ImageURL
	}
	if n.ActionURL != "" {
		doc["action_url"] = n.ActionURL
	}
	if n.Data != nil {
		doc["data"] = n.Data
	}
	if n.ReadAt != nil {
		doc["read_at"] = *n.ReadAt
	}
	if n.DeliveredAt != nil {
		doc["delivered_at"] = n.DeliveredAt
	}
	if len(n.FailedChannels) > 0 {
		doc["failed_channels"] = n.FailedChannels
	}
	if n.TemplateID != "" {
		doc["template_id"] = n.TemplateID
	}
	if n.TemplateData != nil {
		doc["template_data"] = n.TemplateData
	}
	if n.ExpiresAt != nil {
		doc["expires_at"] = *n.ExpiresAt
	}
	if n.TenantID != "" {
		doc["tenant_id"] = n.TenantID
	}

	return doc
}

func (m *MongoStorage) fromDocument(doc bson.M) *adapters.Notification {
	n := &adapters.Notification{}

	if id, ok := doc["_id"].(primitive.ObjectID); ok {
		n.ID = id.Hex()
	}
	if v, ok := doc["recipient_id"].(string); ok {
		n.RecipientID = v
	}
	if v, ok := doc["sender_id"].(string); ok {
		n.SenderID = v
	}
	if v, ok := doc["type"].(string); ok {
		n.Type = v
	}
	if v, ok := doc["title"].(string); ok {
		n.Title = v
	}
	if v, ok := doc["body"].(string); ok {
		n.Body = v
	}
	if v, ok := doc["image_url"].(string); ok {
		n.ImageURL = v
	}
	if v, ok := doc["action_url"].(string); ok {
		n.ActionURL = v
	}
	if v, ok := doc["priority"].(string); ok {
		n.Priority = v
	}
	if v, ok := doc["read"].(bool); ok {
		n.Read = v
	}
	if v, ok := doc["created_at"].(string); ok {
		n.CreatedAt = v
	}
	if v, ok := doc["updated_at"].(string); ok {
		n.UpdatedAt = v
	}

	return n
}
