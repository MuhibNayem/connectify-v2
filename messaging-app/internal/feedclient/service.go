package feedclient

import (
	"context"

	"gitlab.com/spydotech-group/shared-entity/models"
	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
)

// CreatePost calls the gRPC CreatePost method
func (c *Client) CreatePost(ctx context.Context, userID, content, privacy string) (*models.Post, error) {
	resp, err := c.client.CreatePost(ctx, &feedpb.CreatePostRequest{
		UserId:  userID,
		Content: content,
		Privacy: privacy,
	})
	if err != nil {
		return nil, err
	}

	return toModelPost(resp)
}

// GetPost calls the gRPC GetPost method
func (c *Client) GetPost(ctx context.Context, postID string) (*models.Post, error) {
	resp, err := c.client.GetPost(ctx, &feedpb.GetPostRequest{
		PostId: postID,
	})
	if err != nil {
		return nil, err
	}

	return toModelPost(resp)
}

// ListPosts calls the gRPC ListPosts method
func (c *Client) ListPosts(ctx context.Context, viewerID string, page, limit int64) ([]models.Post, error) {
	resp, err := c.client.ListPosts(ctx, &feedpb.ListPostsRequest{
		ViewerId: viewerID,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	posts, _ := toModelFeedResponse(resp)
	return posts, nil
}

// GetPostsByHashtag calls the gRPC GetPostsByHashtag method
func (c *Client) GetPostsByHashtag(ctx context.Context, viewerID, hashtag string, page, limit int64) ([]models.Post, error) {
	resp, err := c.client.GetPostsByHashtag(ctx, &feedpb.GetPostsByHashtagRequest{
		ViewerId: viewerID,
		Hashtag:  hashtag,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	posts, _ := toModelFeedResponse(resp)
	return posts, nil
}

// UpdatePost calls the gRPC UpdatePost method
func (c *Client) UpdatePost(ctx context.Context, postID, userID, content, privacy string) (*models.Post, error) {
	resp, err := c.client.UpdatePost(ctx, &feedpb.UpdatePostRequest{
		PostId:  postID,
		UserId:  userID,
		Content: content,
		Privacy: privacy,
	})
	if err != nil {
		return nil, err
	}
	return toModelPost(resp)
}

// DeletePost calls the gRPC DeletePost method
func (c *Client) DeletePost(ctx context.Context, postID, userID string) error {
	_, err := c.client.DeletePost(ctx, &feedpb.DeletePostRequest{
		PostId: postID,
		UserId: userID,
	})
	return err
}

// UpdatePostStatus calls the gRPC UpdatePostStatus method
func (c *Client) UpdatePostStatus(ctx context.Context, postID, userID, status string) error {
	_, err := c.client.UpdatePostStatus(ctx, &feedpb.UpdatePostStatusRequest{
		PostId: postID,
		UserId: userID,
		Status: status,
	})
	return err
}

// ReactToPost calls the gRPC ReactToPost method
func (c *Client) ReactToPost(ctx context.Context, userID, postID, emoji string) error {
	_, err := c.client.ReactToPost(ctx, &feedpb.ReactToPostRequest{
		UserId: userID,
		PostId: postID,
		Emoji:  emoji,
	})
	return err
}

func (c *Client) ReactToComment(ctx context.Context, userID, commentID, emoji string) error {
	_, err := c.client.ReactToComment(ctx, &feedpb.ReactToCommentRequest{
		UserId:    userID,
		CommentId: commentID,
		Emoji:     emoji,
	})
	return err
}

func (c *Client) ReactToReply(ctx context.Context, userID, replyID, emoji string) error {
	_, err := c.client.ReactToReply(ctx, &feedpb.ReactToReplyRequest{
		UserId:  userID,
		ReplyId: replyID,
		Emoji:   emoji,
	})
	return err
}

// ----------------------------- Comments -----------------------------

func (c *Client) CreateComment(ctx context.Context, userID, postID, content string) (*models.Comment, error) {
	resp, err := c.client.CreateComment(ctx, &feedpb.CreateCommentRequest{
		UserId:  userID,
		PostId:  postID,
		Content: content,
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(resp)
}

func (c *Client) ListComments(ctx context.Context, postID string, page, limit int64) ([]models.Comment, error) {
	resp, err := c.client.ListComments(ctx, &feedpb.ListCommentsRequest{
		PostId: postID,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	var comments []models.Comment
	for _, pbComment := range resp.GetComments() {
		if comment, err := toModelComment(pbComment); err == nil && comment != nil {
			comments = append(comments, *comment)
		}
	}
	return comments, nil
}

func (c *Client) GetComment(ctx context.Context, commentID string) (*models.Comment, error) {
	resp, err := c.client.GetComment(ctx, &feedpb.GetCommentRequest{
		CommentId: commentID,
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(resp)
}

func (c *Client) UpdateComment(ctx context.Context, commentID, userID, content string) (*models.Comment, error) {
	resp, err := c.client.UpdateComment(ctx, &feedpb.UpdateCommentRequest{
		CommentId: commentID,
		UserId:    userID,
		Content:   content,
	})
	if err != nil {
		return nil, err
	}
	return toModelComment(resp)
}

func (c *Client) DeleteComment(ctx context.Context, postID, commentID, userID string) error {
	_, err := c.client.DeleteComment(ctx, &feedpb.DeleteCommentRequest{
		PostId:    postID,
		CommentId: commentID,
		UserId:    userID,
	})
	return err
}

// ----------------------------- Replies -----------------------------

func (c *Client) CreateReply(ctx context.Context, userID, commentID, content string) (*models.Reply, error) {
	resp, err := c.client.CreateReply(ctx, &feedpb.CreateReplyRequest{
		UserId:    userID,
		CommentId: commentID,
		Content:   content,
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(resp)
}

func (c *Client) ListReplies(ctx context.Context, commentID string, page, limit int64) ([]models.Reply, error) {
	resp, err := c.client.ListReplies(ctx, &feedpb.ListRepliesRequest{
		CommentId: commentID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	var replies []models.Reply
	for _, pbReply := range resp.GetReplies() {
		if reply, err := toModelReply(pbReply); err == nil && reply != nil {
			replies = append(replies, *reply)
		}
	}
	return replies, nil
}

func (c *Client) GetReply(ctx context.Context, replyID string) (*models.Reply, error) {
	resp, err := c.client.GetReply(ctx, &feedpb.GetReplyRequest{
		ReplyId: replyID,
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(resp)
}

func (c *Client) UpdateReply(ctx context.Context, replyID, userID, content string) (*models.Reply, error) {
	resp, err := c.client.UpdateReply(ctx, &feedpb.UpdateReplyRequest{
		ReplyId: replyID,
		UserId:  userID,
		Content: content,
	})
	if err != nil {
		return nil, err
	}
	return toModelReply(resp)
}

func (c *Client) DeleteReply(ctx context.Context, commentID, replyID, userID string) error {
	_, err := c.client.DeleteReply(ctx, &feedpb.DeleteReplyRequest{
		CommentId: commentID,
		ReplyId:   replyID,
		UserId:    userID,
	})
	return err
}

// ----------------------------- Albums -----------------------------

func (c *Client) CreateAlbum(ctx context.Context, userID, title, description, privacy string) (*models.Album, error) {
	resp, err := c.client.CreateAlbum(ctx, &feedpb.CreateAlbumRequest{
		UserId:      userID,
		Title:       title,
		Description: description,
		Privacy:     privacy,
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(resp)
}

func (c *Client) GetAlbum(ctx context.Context, albumID string) (*models.Album, error) {
	resp, err := c.client.GetAlbum(ctx, &feedpb.GetAlbumRequest{
		AlbumId: albumID,
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(resp)
}

func (c *Client) UpdateAlbum(ctx context.Context, albumID, title, description, privacy string) (*models.Album, error) {
	resp, err := c.client.UpdateAlbum(ctx, &feedpb.UpdateAlbumRequest{
		AlbumId:     albumID,
		Title:       title,
		Description: description,
		Privacy:     privacy,
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbum(resp)
}

func (c *Client) DeleteAlbum(ctx context.Context, albumID string) error {
	_, err := c.client.DeleteAlbum(ctx, &feedpb.DeleteAlbumRequest{
		AlbumId: albumID,
	})
	return err
}

func (c *Client) ListAlbums(ctx context.Context, userID string, page, limit int64) ([]models.Album, error) {
	resp, err := c.client.ListAlbums(ctx, &feedpb.ListAlbumsRequest{
		UserId: userID,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

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
	resp, err := c.client.AddMediaToAlbum(ctx, &feedpb.AddMediaToAlbumRequest{
		AlbumId:     albumID,
		Url:         url,
		Type:        mediaType,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	return toModelAlbumMedia(resp)
}

func (c *Client) RemoveMediaFromAlbum(ctx context.Context, mediaID string) error {
	_, err := c.client.RemoveMediaFromAlbum(ctx, &feedpb.RemoveMediaFromAlbumRequest{
		MediaId: mediaID,
	})
	return err
}

func (c *Client) GetAlbumMedia(ctx context.Context, albumID string, page, limit int64) ([]models.AlbumMedia, error) {
	resp, err := c.client.GetAlbumMedia(ctx, &feedpb.GetAlbumMediaRequest{
		AlbumId: albumID,
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}

	var mediaList []models.AlbumMedia
	for _, pbMedia := range resp.GetMedia() {
		if media, err := toModelAlbumMedia(pbMedia); err == nil && media != nil {
			mediaList = append(mediaList, *media)
		}
	}
	return mediaList, nil
}
