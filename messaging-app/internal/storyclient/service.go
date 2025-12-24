package storyclient

import (
	"context"

	"gitlab.com/spydotech-group/shared-entity/models"
	storypb "gitlab.com/spydotech-group/shared-entity/proto/story/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateStory creates a new story via gRPC
func (c *Client) CreateStory(ctx context.Context, userID primitive.ObjectID, req *models.CreateStoryRequest) (*models.Story, error) {
	// Convert allowed viewers to strings
	allowedViewers := make([]string, 0, len(req.AllowedViewers))
	for _, id := range req.AllowedViewers {
		allowedViewers = append(allowedViewers, id.Hex())
	}

	// Convert blocked viewers to strings
	blockedViewers := make([]string, 0, len(req.BlockedViewers))
	for _, id := range req.BlockedViewers {
		blockedViewers = append(blockedViewers, id.Hex())
	}

	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateStory(ctx, &storypb.CreateStoryRequest{
			UserId:         userID.Hex(),
			MediaUrl:       req.MediaURL,
			MediaType:      req.MediaType,
			Privacy:        string(req.Privacy),
			AllowedViewers: allowedViewers,
			BlockedViewers: blockedViewers,
		})
	})
	if err != nil {
		return nil, err
	}

	return ToModelStory(result.(*storypb.StoryResponse).Story), nil
}

// GetStory retrieves a story by ID
func (c *Client) GetStory(ctx context.Context, storyID primitive.ObjectID) (*models.Story, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetStory(ctx, &storypb.GetStoryRequest{
			StoryId: storyID.Hex(),
		})
	})
	if err != nil {
		return nil, err
	}

	return ToModelStory(result.(*storypb.StoryResponse).Story), nil
}

// DeleteStory deletes a story
func (c *Client) DeleteStory(ctx context.Context, storyID, userID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteStory(ctx, &storypb.DeleteStoryRequest{
			StoryId: storyID.Hex(),
			UserId:  userID.Hex(),
		})
	})
	return err
}

// GetStoriesFeed returns the stories feed for a user
func (c *Client) GetStoriesFeed(ctx context.Context, userID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int) ([]models.Story, error) {
	// Convert friend IDs to strings
	friendIDStrs := make([]string, 0, len(friendIDs))
	for _, id := range friendIDs {
		friendIDStrs = append(friendIDStrs, id.Hex())
	}

	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetStoriesFeed(ctx, &storypb.GetStoriesFeedRequest{
			UserId:    userID.Hex(),
			FriendIds: friendIDStrs,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
	})
	if err != nil {
		return nil, err
	}

	return ToModelStories(result.(*storypb.StoriesFeedResponse).Stories), nil
}

// GetUserStories returns all active stories for a user
func (c *Client) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetUserStories(ctx, &storypb.GetUserStoriesRequest{
			UserId: userID.Hex(),
		})
	})
	if err != nil {
		return nil, err
	}

	return ToModelStories(result.(*storypb.StoriesResponse).Stories), nil
}

// RecordView records a view on a story
func (c *Client) RecordView(ctx context.Context, storyID, viewerID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.RecordView(ctx, &storypb.RecordViewRequest{
			StoryId:  storyID.Hex(),
			ViewerId: viewerID.Hex(),
		})
	})
	return err
}

// ReactToStory adds a reaction to a story
func (c *Client) ReactToStory(ctx context.Context, storyID, userID primitive.ObjectID, reactionType string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToStory(ctx, &storypb.ReactToStoryRequest{
			StoryId:      storyID.Hex(),
			UserId:       userID.Hex(),
			ReactionType: reactionType,
		})
	})
	return err
}

// GetStoryViewers returns viewers with their reactions for a story
func (c *Client) GetStoryViewers(ctx context.Context, storyID, userID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetStoryViewers(ctx, &storypb.GetStoryViewersRequest{
			StoryId: storyID.Hex(),
			UserId:  userID.Hex(),
		})
	})
	if err != nil {
		return nil, err
	}

	return ToModelStoryViewers(result.(*storypb.StoryViewersResponse).Viewers), nil
}
