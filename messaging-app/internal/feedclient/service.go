package feedclient

import (
	"context"

	"gitlab.com/spydotech-group/shared-entity/models"
	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
)

// CreatePost calls the gRPC CreatePost method
func (c *Client) CreatePost(ctx context.Context, userID, content, privacy string) (*models.Post, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreatePost(ctx, &feedpb.CreatePostRequest{
			UserId:  userID,
			Content: content,
			Privacy: privacy,
		})
	})
	if err != nil {
		return nil, err
	}

	return toModelPost(result.(*feedpb.PostResponse))
}

// GetPost calls the gRPC GetPost method
func (c *Client) GetPost(ctx context.Context, postID string) (*models.Post, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetPost(ctx, &feedpb.GetPostRequest{
			PostId: postID,
		})
	})
	if err != nil {
		return nil, err
	}

	return toModelPost(result.(*feedpb.PostResponse))
}

// ListPosts calls the gRPC ListPosts method
func (c *Client) ListPosts(ctx context.Context, viewerID string, page, limit int64) ([]models.Post, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ListPosts(ctx, &feedpb.ListPostsRequest{
			ViewerId: viewerID,
			Page:     page,
			Limit:    limit,
		})
	})
	if err != nil {
		return nil, err
	}

	posts, _ := toModelFeedResponse(result.(*feedpb.FeedResponse))
	return posts, nil
}

// GetPostsByHashtag calls the gRPC GetPostsByHashtag method
func (c *Client) GetPostsByHashtag(ctx context.Context, viewerID, hashtag string, page, limit int64) ([]models.Post, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetPostsByHashtag(ctx, &feedpb.GetPostsByHashtagRequest{
			ViewerId: viewerID,
			Hashtag:  hashtag,
			Page:     page,
			Limit:    limit,
		})
	})
	if err != nil {
		return nil, err
	}

	posts, _ := toModelFeedResponse(result.(*feedpb.FeedResponse))
	return posts, nil
}

// UpdatePost calls the gRPC UpdatePost method
func (c *Client) UpdatePost(ctx context.Context, postID, userID, content, privacy string) (*models.Post, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdatePost(ctx, &feedpb.UpdatePostRequest{
			PostId:  postID,
			UserId:  userID,
			Content: content,
			Privacy: privacy,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelPost(result.(*feedpb.PostResponse))
}

// DeletePost calls the gRPC DeletePost method
func (c *Client) DeletePost(ctx context.Context, postID, userID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeletePost(ctx, &feedpb.DeletePostRequest{
			PostId: postID,
			UserId: userID,
		})
	})
	return err
}

// UpdatePostStatus calls the gRPC UpdatePostStatus method
func (c *Client) UpdatePostStatus(ctx context.Context, postID, userID, status string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdatePostStatus(ctx, &feedpb.UpdatePostStatusRequest{
			PostId: postID,
			UserId: userID,
			Status: status,
		})
	})
	return err
}

// ReactToPost calls the gRPC ReactToPost method
func (c *Client) ReactToPost(ctx context.Context, userID, postID, emoji string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToPost(ctx, &feedpb.ReactToPostRequest{
			UserId: userID,
			PostId: postID,
			Emoji:  emoji,
		})
	})
	return err
}

func (c *Client) ReactToComment(ctx context.Context, userID, commentID, emoji string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToComment(ctx, &feedpb.ReactToCommentRequest{
			UserId:    userID,
			CommentId: commentID,
			Emoji:     emoji,
		})
	})
	return err
}

func (c *Client) ReactToReply(ctx context.Context, userID, replyID, emoji string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToReply(ctx, &feedpb.ReactToReplyRequest{
			UserId:  userID,
			ReplyId: replyID,
			Emoji:   emoji,
		})
	})
	return err
}

// ----------------------------- Comments -----------------------------

func (c *Client) CreateComment(ctx context.Context, userID, postID, content string) (*models.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateComment(ctx, &feedpb.CreateCommentRequest{
			UserId:  userID,
			PostId:  postID,
			Content: content,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(result.(*feedpb.CommentResponse))
}

func (c *Client) ListComments(ctx context.Context, postID string, page, limit int64) ([]models.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ListComments(ctx, &feedpb.ListCommentsRequest{
			PostId: postID,
			Page:   page,
			Limit:  limit,
		})
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*feedpb.ListCommentsResponse)
	var comments []models.Comment
	for _, pbComment := range resp.GetComments() {
		if comment, err := toModelComment(pbComment); err == nil && comment != nil {
			comments = append(comments, *comment)
		}
	}
	return comments, nil
}

func (c *Client) GetComment(ctx context.Context, commentID string) (*models.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetComment(ctx, &feedpb.GetCommentRequest{
			CommentId: commentID,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(result.(*feedpb.CommentResponse))
}

func (c *Client) UpdateComment(ctx context.Context, commentID, userID, content string) (*models.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdateComment(ctx, &feedpb.UpdateCommentRequest{
			CommentId: commentID,
			UserId:    userID,
			Content:   content,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(result.(*feedpb.CommentResponse))
}

func (c *Client) DeleteComment(ctx context.Context, postID, commentID, userID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteComment(ctx, &feedpb.DeleteCommentRequest{
			PostId:    postID,
			CommentId: commentID,
			UserId:    userID,
		})
	})
	return err
}

// ----------------------------- Replies -----------------------------

func (c *Client) CreateReply(ctx context.Context, userID, commentID, content string) (*models.Reply, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateReply(ctx, &feedpb.CreateReplyRequest{
			UserId:    userID,
			CommentId: commentID,
			Content:   content,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(result.(*feedpb.ReplyResponse))
}

func (c *Client) ListReplies(ctx context.Context, commentID string, page, limit int64) ([]models.Reply, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ListReplies(ctx, &feedpb.ListRepliesRequest{
			CommentId: commentID,
			Page:      page,
			Limit:     limit,
		})
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*feedpb.ListRepliesResponse)
	var replies []models.Reply
	for _, pbReply := range resp.GetReplies() {
		if reply, err := toModelReply(pbReply); err == nil && reply != nil {
			replies = append(replies, *reply)
		}
	}
	return replies, nil
}

func (c *Client) GetReply(ctx context.Context, replyID string) (*models.Reply, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetReply(ctx, &feedpb.GetReplyRequest{
			ReplyId: replyID,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(result.(*feedpb.ReplyResponse))
}

func (c *Client) UpdateReply(ctx context.Context, replyID, userID, content string) (*models.Reply, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdateReply(ctx, &feedpb.UpdateReplyRequest{
			ReplyId: replyID,
			UserId:  userID,
			Content: content,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(result.(*feedpb.ReplyResponse))
}

func (c *Client) DeleteReply(ctx context.Context, commentID, replyID, userID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteReply(ctx, &feedpb.DeleteReplyRequest{
			CommentId: commentID,
			ReplyId:   replyID,
			UserId:    userID,
		})
	})
	return err
}

// ----------------------------- Albums -----------------------------

func (c *Client) CreateAlbum(ctx context.Context, userID, title, description, privacy string) (*models.Album, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateAlbum(ctx, &feedpb.CreateAlbumRequest{
			UserId:      userID,
			Title:       title,
			Description: description,
			Privacy:     privacy,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(result.(*feedpb.AlbumResponse))
}

func (c *Client) GetAlbum(ctx context.Context, albumID string) (*models.Album, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetAlbum(ctx, &feedpb.GetAlbumRequest{
			AlbumId: albumID,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(result.(*feedpb.AlbumResponse))
}

func (c *Client) UpdateAlbum(ctx context.Context, albumID, title, description, privacy string) (*models.Album, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdateAlbum(ctx, &feedpb.UpdateAlbumRequest{
			AlbumId:     albumID,
			Title:       title,
			Description: description,
			Privacy:     privacy,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(result.(*feedpb.AlbumResponse))
}

func (c *Client) DeleteAlbum(ctx context.Context, albumID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteAlbum(ctx, &feedpb.DeleteAlbumRequest{
			AlbumId: albumID,
		})
	})
	return err
}

func (c *Client) ListAlbums(ctx context.Context, userID string, page, limit int64) ([]models.Album, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ListAlbums(ctx, &feedpb.ListAlbumsRequest{
			UserId: userID,
			Page:   page,
			Limit:  limit,
		})
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*feedpb.ListAlbumsResponse)
	var albums []models.Album
	for _, pbAlbum := range resp.GetAlbums() {
		if album, err := toModelAlbum(pbAlbum); err == nil && album != nil {
			albums = append(albums, *album)
		}
	}
	return albums, nil
}

// ----------------------------- Album Media -----------------------------

func (c *Client) AddMediaToAlbum(ctx context.Context, albumID, url, mediaType, description string) (*models.AlbumMedia, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.AddMediaToAlbum(ctx, &feedpb.AddMediaToAlbumRequest{
			AlbumId:     albumID,
			Url:         url,
			Type:        mediaType,
			Description: description,
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbumMedia(result.(*feedpb.AlbumMediaResponse))
}

func (c *Client) RemoveMediaFromAlbum(ctx context.Context, mediaID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.RemoveMediaFromAlbum(ctx, &feedpb.RemoveMediaFromAlbumRequest{
			MediaId: mediaID,
		})
	})
	return err
}

func (c *Client) GetAlbumMedia(ctx context.Context, albumID string, page, limit int64) ([]models.AlbumMedia, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetAlbumMedia(ctx, &feedpb.GetAlbumMediaRequest{
			AlbumId: albumID,
			Page:    page,
			Limit:   limit,
		})
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*feedpb.GetAlbumMediaResponse)
	var mediaList []models.AlbumMedia
	for _, pbMedia := range resp.GetMedia() {
		if media, err := toModelAlbumMedia(pbMedia); err == nil && media != nil {
			mediaList = append(mediaList, *media)
		}
	}
	return mediaList, nil
}
