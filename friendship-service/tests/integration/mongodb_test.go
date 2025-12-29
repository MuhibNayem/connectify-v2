package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testDB     *mongo.Database
	testClient *mongo.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx,
		testcontainers.WithImage("mongo:7.0"),
	)
	if err != nil {
		log.Fatalf("Failed to start MongoDB container: %v", err)
	}

	defer func() {
		if err := mongoContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate MongoDB container: %v", err)
		}
	}()

	// Get connection string
	uri, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	// Connect to MongoDB
	testClient, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	testDB = testClient.Database("friendship_test")

	// Run tests
	code := m.Run()

	// Cleanup
	_ = testClient.Disconnect(ctx)

	os.Exit(code)
}

func TestFriendshipRepository_CreateRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("friendships")

	// Clean up before test
	_, _ = collection.DeleteMany(ctx, bson.M{})

	requesterID := primitive.NewObjectID()
	receiverID := primitive.NewObjectID()

	// Create friendship
	friendship := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := collection.InsertOne(ctx, friendship)
	if err != nil {
		t.Fatalf("Failed to insert friendship: %v", err)
	}

	// Verify insertion
	if result.InsertedID == nil {
		t.Error("Expected InsertedID to be non-nil")
	}

	// Read back
	var retrieved models.Friendship
	err = collection.FindOne(ctx, bson.M{"_id": friendship.ID}).Decode(&retrieved)
	if err != nil {
		t.Fatalf("Failed to retrieve friendship: %v", err)
	}

	if retrieved.Status != models.FriendshipStatusPending {
		t.Errorf("Expected status %s, got %s", models.FriendshipStatusPending, retrieved.Status)
	}
	if retrieved.RequesterID != requesterID {
		t.Errorf("Expected requesterID %s, got %s", requesterID.Hex(), retrieved.RequesterID.Hex())
	}
}

func TestFriendshipRepository_AcceptRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("friendships")

	// Create pending friendship
	friendship := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: primitive.NewObjectID(),
		ReceiverID:  primitive.NewObjectID(),
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, friendship)
	if err != nil {
		t.Fatalf("Failed to insert friendship: %v", err)
	}

	// Update to accepted
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": friendship.ID},
		bson.M{
			"$set": bson.M{
				"status":     models.FriendshipStatusAccepted,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to update friendship: %v", err)
	}

	// Verify update
	var retrieved models.Friendship
	err = collection.FindOne(ctx, bson.M{"_id": friendship.ID}).Decode(&retrieved)
	if err != nil {
		t.Fatalf("Failed to retrieve friendship: %v", err)
	}

	if retrieved.Status != models.FriendshipStatusAccepted {
		t.Errorf("Expected status %s, got %s", models.FriendshipStatusAccepted, retrieved.Status)
	}
}

func TestFriendshipRepository_Unfriend(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("friendships")

	// Create accepted friendship
	friendship := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: primitive.NewObjectID(),
		ReceiverID:  primitive.NewObjectID(),
		Status:      models.FriendshipStatusAccepted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, friendship)
	if err != nil {
		t.Fatalf("Failed to insert friendship: %v", err)
	}

	// Delete friendship
	result, err := collection.DeleteOne(ctx, bson.M{"_id": friendship.ID})
	if err != nil {
		t.Fatalf("Failed to delete friendship: %v", err)
	}

	if result.DeletedCount != 1 {
		t.Errorf("Expected 1 deleted, got %d", result.DeletedCount)
	}

	// Verify deletion
	err = collection.FindOne(ctx, bson.M{"_id": friendship.ID}).Decode(&models.Friendship{})
	if err != mongo.ErrNoDocuments {
		t.Errorf("Expected ErrNoDocuments, got %v", err)
	}
}

func TestFriendshipRepository_DuplicateRequest_Fails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("friendships_unique")

	// Create unique index
	_, _ = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "requester_id", Value: 1},
			{Key: "receiver_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})

	requesterID := primitive.NewObjectID()
	receiverID := primitive.NewObjectID()

	// First insertion should succeed
	friendship1 := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, friendship1)
	if err != nil {
		t.Fatalf("First insert should succeed: %v", err)
	}

	// Second insertion with same users should fail
	friendship2 := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      models.FriendshipStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = collection.InsertOne(ctx, friendship2)
	if err == nil {
		t.Error("Expected duplicate key error, got nil")
	}
}

func TestFriendshipRepository_AreFriends(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("friendships")

	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()

	// Initially not friends
	count, err := collection.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"requester_id": userA, "receiver_id": userB, "status": models.FriendshipStatusAccepted},
			{"requester_id": userB, "receiver_id": userA, "status": models.FriendshipStatusAccepted},
		},
	})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 friends initially, got %d", count)
	}

	// Create accepted friendship
	friendship := &models.Friendship{
		ID:          primitive.NewObjectID(),
		RequesterID: userA,
		ReceiverID:  userB,
		Status:      models.FriendshipStatusAccepted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, _ = collection.InsertOne(ctx, friendship)

	// Now should be friends
	count, err = collection.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"requester_id": userA, "receiver_id": userB, "status": models.FriendshipStatusAccepted},
			{"requester_id": userB, "receiver_id": userA, "status": models.FriendshipStatusAccepted},
		},
	})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 friendship, got %d", count)
	}
}

func TestBlockRepository_BlockUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("blocks")

	blockerID := primitive.NewObjectID()
	blockedID := primitive.NewObjectID()

	block := bson.M{
		"_id":        primitive.NewObjectID(),
		"blocker_id": blockerID,
		"blocked_id": blockedID,
		"created_at": time.Now(),
	}

	_, err := collection.InsertOne(ctx, block)
	if err != nil {
		t.Fatalf("Failed to insert block: %v", err)
	}

	// Verify block exists
	count, err := collection.CountDocuments(ctx, bson.M{
		"blocker_id": blockerID,
		"blocked_id": blockedID,
	})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 block, got %d", count)
	}
}

func TestOutboxRepository_CreateAndProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("outbox_events")

	event := bson.M{
		"_id":          primitive.NewObjectID(),
		"aggregate_id": primitive.NewObjectID(),
		"event_type":   "FriendRequestSent",
		"payload": bson.M{
			"requester_id": primitive.NewObjectID().Hex(),
			"receiver_id":  primitive.NewObjectID().Hex(),
		},
		"status":      "pending",
		"retry_count": 0,
		"max_retries": 5,
		"next_retry":  time.Now(),
		"created_at":  time.Now(),
	}

	_, err := collection.InsertOne(ctx, event)
	if err != nil {
		t.Fatalf("Failed to insert outbox event: %v", err)
	}

	// Get pending events
	cursor, err := collection.Find(ctx, bson.M{"status": "pending"})
	if err != nil {
		t.Fatalf("Failed to find pending events: %v", err)
	}
	defer cursor.Close(ctx)

	var events []bson.M
	if err := cursor.All(ctx, &events); err != nil {
		t.Fatalf("Failed to decode events: %v", err)
	}

	if len(events) < 1 {
		t.Error("Expected at least 1 pending event")
	}

	// Mark as processed
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": event["_id"]},
		bson.M{
			"$set": bson.M{
				"status":       "processed",
				"processed_at": time.Now(),
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to mark as processed: %v", err)
	}
}

func TestVersionedWrites_OptimisticLocking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("versioned_friendships")

	// Create friendship with version
	friendship := bson.M{
		"_id":          primitive.NewObjectID(),
		"requester_id": primitive.NewObjectID(),
		"receiver_id":  primitive.NewObjectID(),
		"status":       "pending",
		"version":      int64(1),
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}

	_, err := collection.InsertOne(ctx, friendship)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Update with correct version should succeed
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": friendship["_id"], "version": int64(1)},
		bson.M{
			"$set": bson.M{"status": "accepted", "updated_at": time.Now()},
			"$inc": bson.M{"version": 1},
		},
	)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}
	if result.MatchedCount != 1 {
		t.Error("Expected update to match 1 document")
	}

	// Update with stale version should fail
	result, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": friendship["_id"], "version": int64(1)}, // Stale version
		bson.M{
			"$set": bson.M{"status": "rejected", "updated_at": time.Now()},
			"$inc": bson.M{"version": 1},
		},
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if result.MatchedCount != 0 {
		t.Error("Expected stale version update to match 0 documents")
	}
}

func TestConcurrentModifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	collection := testDB.Collection("concurrent_test")

	// Create document
	doc := bson.M{
		"_id":     primitive.NewObjectID(),
		"counter": 0,
		"version": int64(0),
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Simulate concurrent updates
	successCount := 0
	iterations := 10

	for i := 0; i < iterations; i++ {
		// Read current version
		var current bson.M
		err := collection.FindOne(ctx, bson.M{"_id": doc["_id"]}).Decode(&current)
		if err != nil {
			continue
		}

		version, ok := current["version"].(int64)
		if !ok {
			version = int64(current["version"].(int32))
		}

		// Try to update
		result, err := collection.UpdateOne(
			ctx,
			bson.M{"_id": doc["_id"], "version": version},
			bson.M{
				"$inc": bson.M{"counter": 1, "version": 1},
			},
		)
		if err == nil && result.MatchedCount == 1 {
			successCount++
		}
	}

	fmt.Printf("Successful updates: %d/%d\n", successCount, iterations)

	// Final counter should equal success count
	var final bson.M
	_ = collection.FindOne(ctx, bson.M{"_id": doc["_id"]}).Decode(&final)
	counter, ok := final["counter"].(int32)
	if !ok {
		counter = int32(final["counter"].(int64))
	}

	if int(counter) != successCount {
		t.Errorf("Counter mismatch: expected %d, got %d", successCount, counter)
	}
}
