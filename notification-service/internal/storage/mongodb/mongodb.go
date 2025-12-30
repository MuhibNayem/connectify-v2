package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client                  *mongo.Client
	collection              *mongo.Collection
	outboxCollection        *mongo.Collection
	deliveryStateCollection *mongo.Collection
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
	outboxColl := client.Database(database).Collection("outbox")
	deliveryStateColl := client.Database(database).Collection("delivery_states")

	_, err = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "read", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	_, err = outboxColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "timestamp", Value: 1}}},
		{Keys: bson.D{{Key: "locked_until", Value: 1}}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox indexes: %w", err)
	}

	return &MongoStorage{
		client:                  client,
		collection:              coll,
		outboxCollection:        outboxColl,
		deliveryStateCollection: deliveryStateColl,
	}, nil
}

func (m *MongoStorage) Create(ctx context.Context, notification *adapters.Notification) error {
	if notification.ID == "" {
		notification.ID = uuidString()
	}

	now := time.Now().UTC()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)

	doc := m.toDocument(notification)
	_, err := m.collection.InsertOne(ctx, doc)
	return err
}

func (m *MongoStorage) Get(ctx context.Context, id string) (*adapters.Notification, error) {
	var doc bson.M
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, adapters.ErrNotFound
		}
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
	if len(query.Channels) > 0 {
		filter["channels"] = bson.M{"$in": query.Channels}
	}
	if query.Priority != "" {
		filter["priority"] = query.Priority
	}
	if query.TenantID != "" {
		filter["tenant_id"] = query.TenantID
	}
	if query.Since != nil {
		if since, err := time.Parse(time.RFC3339, *query.Since); err == nil {
			filter["created_at"] = bson.M{"$gte": since}
		}
	}
	if query.Until != nil {
		if until, err := time.Parse(time.RFC3339, *query.Until); err == nil {
			if existing, ok := filter["created_at"].(bson.M); ok {
				existing["$lte"] = until
			} else {
				filter["created_at"] = bson.M{"$lte": until}
			}
		}
	}
	if query.Cursor != "" {
		if ts, cursorID, err := parseCursor(query.Cursor); err == nil {
			filter["$or"] = []bson.M{
				{"created_at": bson.M{"$lt": ts}},
				{"created_at": ts, "_id": bson.M{"$lt": cursorID}},
			}
		}
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}

	findOpts := options.Find().
		SetLimit(int64(limit + 1)).
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}})

	cur, err := m.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var notifications []*adapters.Notification
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		notifications = append(notifications, m.fromDocument(doc))
	}

	hasMore := len(notifications) > limit
	var nextCursor string
	if hasMore {
		last := notifications[len(notifications)-1]
		nextCursor = buildCursor(last.CreatedAt, last.ID)
		notifications = notifications[:len(notifications)-1]
	}

	return &adapters.ListResult{
		Notifications: notifications,
		Total:         int64(len(notifications)),
		NextCursor:    nextCursor,
		HasMore:       hasMore,
	}, nil
}

func (m *MongoStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updateDoc := bson.M{
		"updated_at": time.Now().UTC(),
	}
	if read, ok := updates["read"].(bool); ok {
		updateDoc["read"] = read
		if read {
			updateDoc["read_at"] = time.Now().UTC()
		}
	}
	_, err := m.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updateDoc})
	return err
}

func (m *MongoStorage) Delete(ctx context.Context, id string) error {
	_, err := m.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *MongoStorage) GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return m.collection.CountDocuments(ctx, bson.M{
		"recipient_id": recipientID,
		"read":         false,
	})
}

func (m *MongoStorage) BatchMarkAsRead(ctx context.Context, recipientID string, ids []string) (int64, error) {
	update := bson.M{
		"$set": bson.M{
			"read":       true,
			"read_at":    time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		},
	}

	result, err := m.collection.UpdateMany(ctx, bson.M{
		"_id":          bson.M{"$in": ids},
		"recipient_id": recipientID,
	}, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (m *MongoStorage) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := m.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$exists": true, "$lt": time.Now().UTC()},
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
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"notification_id": state.NotificationID}
	update := bson.M{"$set": state}

	_, err := m.deliveryStateCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStorage) GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error) {
	var state models.NotificationDeliveryState
	err := m.deliveryStateCollection.FindOne(ctx, bson.M{"notification_id": notificationID}).Decode(&state)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (m *MongoStorage) CreateWithOutbox(ctx context.Context, notification *adapters.Notification, event *adapters.NotificationEvent) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := m.insertNotification(sessCtx, notification); err != nil {
			return nil, err
		}
		if _, err := m.outboxCollection.InsertOne(sessCtx, event); err != nil {
			return nil, err
		}
		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	return err
}

func (m *MongoStorage) CreateBatchWithOutbox(ctx context.Context, notifications []*adapters.Notification, events []*adapters.NotificationEvent) error {
	if len(notifications) == 0 {
		return nil
	}

	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if len(notifications) > 0 {
			docs := make([]interface{}, len(notifications))
			for i, n := range notifications {
				docs[i] = m.toDocument(n)
			}
			if _, err := m.collection.InsertMany(sessCtx, docs); err != nil {
				return nil, err
			}
		}

		if len(events) > 0 {
			eventDocs := make([]interface{}, len(events))
			for i, e := range events {
				eventDocs[i] = e
			}
			if _, err := m.outboxCollection.InsertMany(sessCtx, eventDocs); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	return err
}

func (m *MongoStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
	var events []*adapters.NotificationEvent
	now := time.Now().UTC()
	lockDuration := 2 * time.Minute
	lockedUntil := now.Add(lockDuration)

	for i := 0; i < limit; i++ {
		filter := bson.M{
			"$or": []bson.M{
				{"locked_until": bson.M{"$exists": false}},
				{"locked_until": nil},
				{"locked_until": bson.M{"$lt": now}},
			},
		}

		update := bson.M{"$set": bson.M{"locked_until": lockedUntil}}
		opts := options.FindOneAndUpdate().
			SetSort(bson.D{{Key: "timestamp", Value: 1}}).
			SetReturnDocument(options.After)

		var event adapters.NotificationEvent
		err := m.outboxCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&event)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break
			}
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

func (m *MongoStorage) DeleteOutboxEvent(ctx context.Context, eventID string) error {
	_, err := m.outboxCollection.DeleteOne(ctx, bson.M{"id": eventID})
	return err
}

func (m *MongoStorage) Close() error {
	return m.client.Disconnect(context.Background())
}

func (m *MongoStorage) insertNotification(ctx context.Context, notification *adapters.Notification) error {
	if notification.ID == "" {
		notification.ID = uuidString()
	}
	now := time.Now().UTC()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)
	_, err := m.collection.InsertOne(ctx, m.toDocument(notification))
	return err
}

func (m *MongoStorage) toDocument(n *adapters.Notification) bson.M {
	createdAt := time.Now().UTC()
	if n.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, n.CreatedAt); err == nil {
			createdAt = parsed
		}
	}

	updatedAt := createdAt
	if n.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, n.UpdatedAt); err == nil {
			updatedAt = parsed
		}
	}

	doc := bson.M{
		"_id":             n.ID,
		"recipient_id":    n.RecipientID,
		"sender_id":       n.SenderID,
		"type":            n.Type,
		"title":           n.Title,
		"body":            n.Body,
		"image_url":       n.ImageURL,
		"action_url":      n.ActionURL,
		"priority":        n.Priority,
		"channels":        n.Channels,
		"data":            n.Data,
		"read":            n.Read,
		"read_at":         parseTimePtr(n.ReadAt),
		"delivered_at":    n.DeliveredAt,
		"failed_channels": n.FailedChannels,
		"template_id":     n.TemplateID,
		"template_data":   n.TemplateData,
		"tenant_id":       n.TenantID,
		"created_at":      createdAt,
		"updated_at":      updatedAt,
	}

	if n.ExpiresAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *n.ExpiresAt); err == nil {
			doc["expires_at"] = parsed
		}
	}

	return doc
}

func (m *MongoStorage) fromDocument(doc bson.M) *adapters.Notification {
	n := &adapters.Notification{}

	if v, ok := doc["_id"].(string); ok {
		n.ID = v
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
	if v, ok := doc["channels"].(bson.A); ok {
		n.Channels = toStringSlice(v)
	}
	if v, ok := doc["data"].(bson.M); ok {
		n.Data = map[string]interface{}(v)
	}
	if v, ok := doc["read"].(bool); ok {
		n.Read = v
	}
	if v, ok := doc["read_at"].(time.Time); ok && !v.IsZero() {
		readAt := v.Format(time.RFC3339)
		n.ReadAt = &readAt
	}
	if v, ok := doc["delivered_at"].(bson.M); ok {
		tmp := map[string]string{}
		for key, value := range v {
			if t, ok := value.(time.Time); ok {
				tmp[key] = t.Format(time.RFC3339)
			}
		}
		n.DeliveredAt = tmp
	}
	if v, ok := doc["failed_channels"].(bson.A); ok {
		n.FailedChannels = toStringSlice(v)
	}
	if v, ok := doc["template_id"].(string); ok {
		n.TemplateID = v
	}
	if v, ok := doc["template_data"].(bson.M); ok {
		n.TemplateData = map[string]interface{}(v)
	}
	if v, ok := doc["tenant_id"].(string); ok {
		n.TenantID = v
	}
	if v, ok := doc["expires_at"].(time.Time); ok && !v.IsZero() {
		str := v.Format(time.RFC3339)
		n.ExpiresAt = &str
	}
	if v, ok := doc["created_at"].(time.Time); ok {
		n.CreatedAt = v.Format(time.RFC3339)
	}
	if v, ok := doc["updated_at"].(time.Time); ok {
		n.UpdatedAt = v.Format(time.RFC3339)
	}

	return n
}

func toStringSlice(arr bson.A) []string {
	result := make([]string, 0, len(arr))
	for _, val := range arr {
		if s, ok := val.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseTimePtr(str *string) interface{} {
	if str == nil || *str == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, *str); err == nil {
		return parsed
	}
	return nil
}

func uuidString() string {
	return uuid.NewString()
}

func buildCursor(createdAt string, id string) string {
	return fmt.Sprintf("%s|%s", createdAt, id)
}

func parseCursor(cursor string) (time.Time, string, error) {
	parts := strings.Split(cursor, "|")
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return ts, parts[1], nil
}
