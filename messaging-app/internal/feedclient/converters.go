package feedclient

import (
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toModelPost(pb *feedpb.PostResponse) (*models.Post, error) {
	if pb == nil {
		return nil, nil
	}

	id, err := primitive.ObjectIDFromHex(pb.GetId())
	if err != nil {
		return nil, err
	}

	userID, err := primitive.ObjectIDFromHex(pb.GetUserId())
	if err != nil {
		// Log error or handle? For now return error as it's critical
		return nil, err
	}

	post := &models.Post{
		ID:        id,
		UserID:    userID,
		Content:   pb.GetContent(),
		Privacy:   models.PrivacySettingType(pb.GetPrivacy()),
		CreatedAt: fromTimestamp(pb.GetCreatedAt()),
		UpdatedAt: fromTimestamp(pb.GetUpdatedAt()),
		// Map other fields as they become available in Proto
		// Media: ...
		// Mentions: ...
		// Hashtags: ...
	}
	return post, nil
}

func toModelFeedResponse(pb *feedpb.FeedResponse) ([]models.Post, int64) {
	if pb == nil {
		return []models.Post{}, 0
	}

	var posts []models.Post
	for _, p := range pb.GetPosts() {
		if post, err := toModelPost(p); err == nil && post != nil {
			posts = append(posts, *post)
		}
	}

	// pb.GetTotal() should be used if available, otherwise len(posts)
	total := pb.GetTotal()
	if total == 0 {
		total = int64(len(posts))
	}

	return posts, total
}

func fromTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func toModelComment(pb *feedpb.CommentResponse) (*models.Comment, error) {
	if pb == nil {
		return nil, nil
	}
	id, err := primitive.ObjectIDFromHex(pb.GetId())
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(pb.GetUserId())
	if err != nil {
		return nil, err
	}
	postID, err := primitive.ObjectIDFromHex(pb.GetPostId())
	if err != nil {
		return nil, err
	}

	return &models.Comment{
		ID:        id,
		PostID:    postID,
		UserID:    userID,
		Content:   pb.GetContent(),
		CreatedAt: fromTimestamp(pb.GetCreatedAt()),
		UpdatedAt: fromTimestamp(pb.GetUpdatedAt()),
	}, nil
}

func toModelReply(pb *feedpb.ReplyResponse) (*models.Reply, error) {
	if pb == nil {
		return nil, nil
	}
	id, err := primitive.ObjectIDFromHex(pb.GetId())
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(pb.GetUserId())
	if err != nil {
		return nil, err
	}
	commentID, err := primitive.ObjectIDFromHex(pb.GetCommentId())
	if err != nil {
		return nil, err
	}

	return &models.Reply{
		ID:        id,
		CommentID: commentID,
		UserID:    userID,
		Content:   pb.GetContent(),
		CreatedAt: fromTimestamp(pb.GetCreatedAt()),
		UpdatedAt: fromTimestamp(pb.GetUpdatedAt()),
	}, nil
}

func toModelAlbum(pb *feedpb.AlbumResponse) (*models.Album, error) {
	if pb == nil {
		return nil, nil
	}
	id, err := primitive.ObjectIDFromHex(pb.GetId())
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(pb.GetUserId())
	if err != nil {
		return nil, err
	}

	return &models.Album{
		ID:          id,
		UserID:      userID,
		Title:       pb.GetTitle(),
		Description: pb.GetDescription(),
		Privacy:     models.PrivacySettingType(pb.GetPrivacy()),
		CreatedAt:   fromTimestamp(pb.GetCreatedAt()),
		UpdatedAt:   fromTimestamp(pb.GetUpdatedAt()),
	}, nil
}

func toModelAlbumMedia(pb *feedpb.AlbumMediaResponse) (*models.AlbumMedia, error) {
	if pb == nil {
		return nil, nil
	}
	id, err := primitive.ObjectIDFromHex(pb.GetId())
	if err != nil {
		return nil, err
	}
	albumID, err := primitive.ObjectIDFromHex(pb.GetAlbumId())
	if err != nil {
		return nil, err
	}

	return &models.AlbumMedia{
		ID:          id,
		AlbumID:     albumID,
		URL:         pb.GetUrl(),
		Type:        pb.GetType(),
		Description: pb.GetDescription(),
		CreatedAt:   fromTimestamp(pb.GetCreatedAt()),
	}, nil
}
