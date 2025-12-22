package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gitlab.com/spydotech-group/feed-service/internal/events"
	"gitlab.com/spydotech-group/feed-service/internal/repository"
	"gitlab.com/spydotech-group/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FeedService struct {
	repo      *repository.FeedRepository
	graphRepo *repository.GraphRepository
	producer  *events.EventProducer
}

func NewFeedService(repo *repository.FeedRepository, graphRepo *repository.GraphRepository, producer *events.EventProducer) *FeedService {
	return &FeedService{
		repo:      repo,
		graphRepo: graphRepo,
		producer:  producer,
	}
}

// CreatePost creates a new post
func (s *FeedService) CreatePost(ctx context.Context, userID string, content string, privacy string) (*models.Post, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	post := &models.Post{
		UserID:    uID,
		Content:   content,
		Privacy:   models.PrivacySettingType(privacy), // Assuming simplified conversion for now
		Status:    models.PostStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// TODO: Handle Media, Mentions, Hashtags parsing
	}

	createdPost, err := s.repo.CreatePost(ctx, post)
	if err != nil {
		return nil, err
	}

	// 3. Publish Event (Smart Producer: Calculate Recipients here)
	// We determine WHO should receive this update so the consumer (messaging-app) doesn't need to query the DB.

	postData, err := json.Marshal(createdPost)
	if err != nil {
		// Should not happen, but log it
		fmt.Printf("Error marshaling post for event: %v\n", err)
	} else {
		var recipientIDs []string
		if post.Privacy == "PUBLIC" || post.Privacy == "FRIENDS" {
			// Fetch friends from Neo4j (Graph Source of Truth)
			friends, err := s.graphRepo.GetFriendIDs(ctx, post.UserID)
			if err != nil {
				// Log error but don't fail the request
				// log.Printf("Failed to get friends: %v", err)
			} else {
				// Friends are already strings
				recipientIDs = append(recipientIDs, friends...)
			}
		}
		// Always include self
		recipientIDs = append(recipientIDs, post.UserID.Hex())

		postCreatedEvent := models.WebSocketEvent{
			Type:       "PostCreated",
			Data:       postData,
			Recipients: recipientIDs,
		}

		if err := s.producer.PublishEvent("messages", postCreatedEvent); err != nil {
			fmt.Printf("Error publishing WS event: %v\n", err)
		}
	}

	// 2. Notifications for Mentions (Mock Logic - needs Mentions parsing)
	// TODO: Parse mentions and send notifications

	return createdPost, nil
}

func (s *FeedService) GetPost(ctx context.Context, postID string) (*models.Post, error) {
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID")
	}
	return s.repo.GetPostByID(ctx, pID)
}

func (s *FeedService) ListPosts(ctx context.Context, viewerID string, page, limit int64) ([]models.Post, error) {
	vID, err := primitive.ObjectIDFromHex(viewerID)
	if err != nil {
		return nil, errors.New("invalid viewer ID")
	}

	// 1. Get Friends List from Neo4j (Graph Source of Truth)
	friendIDTags, err := s.graphRepo.GetFriendIDs(ctx, vID)
	if err != nil {
		return nil, err
	}

	// Convert string IDs to ObjectIDs for Mongo Query
	var friendIDs []primitive.ObjectID
	for _, idStr := range friendIDTags {
		if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
			friendIDs = append(friendIDs, oid)
		}
	}

	// 2. Build Query: Friends Posts (Public/Friends) OR My Posts (All)
	// Optimization: Remove Global Public "Firehose" to match Facebook-style personalized feed.
	filter := bson.M{
		"$or": []bson.M{
			// Friends' Posts: Must be Friends or Public privacy
			{
				"user_id": bson.M{"$in": friendIDs},
				"privacy": bson.M{"$in": []string{"PUBLIC", "FRIENDS"}},
				"status":  "active",
			},
			// My Posts: I can see everything I posted
			{
				"user_id": vID,
				"status":  "active",
			},
		},
	}

	// 3. Pagination is handled by repo (needs proper options construction)
	// For now passing nil options as POC
	return s.repo.ListPosts(ctx, filter, nil)
}

// ... Additional methods (Update, Delete) would go here
