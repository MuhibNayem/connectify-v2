package storyclient

import (
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	storypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/story/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ToModelStory converts a proto Story to a model Story
func ToModelStory(pb *storypb.Story) *models.Story {
	if pb == nil {
		return nil
	}

	id, _ := primitive.ObjectIDFromHex(pb.Id)
	userID, _ := primitive.ObjectIDFromHex(pb.UserId)

	// Convert allowed viewers
	allowedViewers := make([]primitive.ObjectID, 0, len(pb.AllowedViewers))
	for _, idStr := range pb.AllowedViewers {
		if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
			allowedViewers = append(allowedViewers, oid)
		}
	}

	// Convert blocked viewers
	blockedViewers := make([]primitive.ObjectID, 0, len(pb.BlockedViewers))
	for _, idStr := range pb.BlockedViewers {
		if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
			blockedViewers = append(blockedViewers, oid)
		}
	}

	var author models.PostAuthor
	if pb.Author != nil {
		author = models.PostAuthor{
			ID:       pb.Author.Id,
			Username: pb.Author.Username,
			FullName: pb.Author.FullName,
			Avatar:   pb.Author.Avatar,
		}
	}

	story := &models.Story{
		ID:             id,
		UserID:         userID,
		Author:         author,
		MediaURL:       pb.MediaUrl,
		MediaType:      pb.MediaType,
		Privacy:        models.PrivacySettingType(pb.Privacy),
		AllowedViewers: allowedViewers,
		BlockedViewers: blockedViewers,
		ViewCount:      int(pb.ViewCount),
		ReactionCount:  int(pb.ReactionCount),
	}

	if pb.CreatedAt != nil {
		story.CreatedAt = pb.CreatedAt.AsTime()
	}
	if pb.ExpiresAt != nil {
		story.ExpiresAt = pb.ExpiresAt.AsTime()
	}

	return story
}

// ToModelStories converts a slice of proto Stories to model Stories
func ToModelStories(pbs []*storypb.Story) []models.Story {
	stories := make([]models.Story, 0, len(pbs))
	for _, pb := range pbs {
		if story := ToModelStory(pb); story != nil {
			stories = append(stories, *story)
		}
	}
	return stories
}

// ToModelStoryViewer converts a proto StoryViewer to a model StoryViewerResponse
func ToModelStoryViewer(pb *storypb.StoryViewer) *models.StoryViewerResponse {
	if pb == nil {
		return nil
	}

	userID, _ := primitive.ObjectIDFromHex(pb.User.Id)

	viewer := &models.StoryViewerResponse{
		User: models.UserShortResponse{
			ID:       userID,
			Username: pb.User.Username,
			FullName: pb.User.FullName,
			Avatar:   pb.User.Avatar,
		},
		ReactionType: pb.ReactionType,
	}

	if pb.ViewedAt != nil {
		viewer.ViewedAt = pb.ViewedAt.AsTime()
	}

	return viewer
}

// ToModelStoryViewers converts a slice of proto StoryViewers to model StoryViewerResponses
func ToModelStoryViewers(pbs []*storypb.StoryViewer) []models.StoryViewerResponse {
	viewers := make([]models.StoryViewerResponse, 0, len(pbs))
	for _, pb := range pbs {
		if viewer := ToModelStoryViewer(pb); viewer != nil {
			viewers = append(viewers, *viewer)
		}
	}
	return viewers
}
