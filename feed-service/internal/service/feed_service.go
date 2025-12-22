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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FeedService struct {
	repo      *repository.FeedRepository
	cacheRepo *repository.CacheRepository
	graphRepo *repository.GraphRepository
	producer  *events.EventProducer
}

func NewFeedService(repo *repository.FeedRepository, cacheRepo *repository.CacheRepository, graphRepo *repository.GraphRepository, producer *events.EventProducer) *FeedService {
	return &FeedService{
		repo:      repo,
		cacheRepo: cacheRepo,
		graphRepo: graphRepo,
		producer:  producer,
	}
}

// UpdatePostStatus updates the status of a post
func (s *FeedService) UpdatePostStatus(ctx context.Context, postID, userID, status string) error {
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.New("invalid post ID")
	}
	// Verify ownership or admin rights (logic simplified for now)
	return s.repo.UpdatePostStatus(ctx, pID, status)
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
	// 1. Try Cache
	cachedPost, err := s.cacheRepo.GetPost(ctx, postID)
	if err == nil && cachedPost != nil {
		return cachedPost, nil
	}

	// 2. Fallback to DB
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID")
	}
	post, err := s.repo.GetPostByID(ctx, pID)
	if err != nil {
		return nil, err
	}

	// 3. Set Cache (Async or Blocking? Blocking is safer for consistency)
	_ = s.cacheRepo.SetPost(ctx, post)

	return post, nil
}

func (s *FeedService) UpdatePost(ctx context.Context, postID, userID, content, privacy string) (*models.Post, error) {
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID")
	}
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// 1. Verify Ownership (Repo update uses filter, but good to check if we want specific error)
	// For now, let's rely on Repo's atomic update or check if exists first.
	// Repo UpdatePost takes ID only. We should probably verify ownership here.
	existing, err := s.repo.GetPostByID(ctx, pID)
	if err != nil {
		return nil, err
	}
	if existing.UserID != uID {
		return nil, errors.New("unauthorized")
	}

	// 2. Prepare Update
	update := bson.M{}
	if content != "" {
		update["content"] = content
	}
	if privacy != "" {
		update["privacy"] = privacy
	}

	updatedPost, err := s.repo.UpdatePost(ctx, pID, update)
	if err != nil {
		return nil, err
	}

	// Invalidate Cache
	_ = s.cacheRepo.InvalidatePost(ctx, postID)

	// 3. Publish Event
	postData, _ := json.Marshal(updatedPost)
	s.producer.PublishEvent("messages", models.WebSocketEvent{
		Type:       "PostUpdated",
		Data:       postData,
		Recipients: []string{userID}, // Simplified
	})

	return updatedPost, nil
}

func (s *FeedService) GetPostsByHashtag(ctx context.Context, viewerID string, hashtag string, page, limit int64) ([]models.Post, error) {
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// In a real app, we might also filter by blocked users or more complex privacy,
	// but mostly hashtags are for public discovery.
	return s.repo.GetPostsByHashtag(ctx, hashtag, limit, offset)
}

func (s *FeedService) DeletePost(ctx context.Context, postID, userID string) error {
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.New("invalid post ID")
	}
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	err = s.repo.DeletePost(ctx, uID, pID)
	if err != nil {
		return err
	}

	// Publish Event
	s.producer.PublishEvent("messages", models.WebSocketEvent{
		Type:       "PostDeleted",
		Data:       []byte(fmt.Sprintf(`{"id": "%s"}`, postID)),
		Recipients: []string{userID},
	})

	return nil
}

func (s *FeedService) ListPosts(ctx context.Context, viewerID string, page, limit int64) ([]models.Post, error) {
	vID, err := primitive.ObjectIDFromHex(viewerID)
	if err != nil {
		return nil, errors.New("invalid viewer ID")
	}

	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 1. Try Fetching from Redis Timeline
	timelineIDs, err := s.cacheRepo.GetTimeline(ctx, viewerID, offset, limit)
	if err == nil && len(timelineIDs) > 0 {
		// Found in Redis. Hydrate posts.
		posts, missingIDs, err := s.cacheRepo.GetPosts(ctx, timelineIDs)
		if err == nil {
			// If we have some missing posts (evicted?), we could fetch them from DB
			if len(missingIDs) > 0 {
				fmt.Printf("Cache partial miss for %d posts\n", len(missingIDs))
				// Optionally fetch missing ones from DB and re-cache
				for _, mid := range missingIDs {
					if oid, err := primitive.ObjectIDFromHex(mid); err == nil {
						p, err := s.repo.GetPostByID(ctx, oid)
						if err == nil {
							posts = append(posts, p)
							_ = s.cacheRepo.SetPost(ctx, p)
						}
					}
				}
			}

			// Dereference pointers to values (sort order corresponds to timelineIDs order, ideally)
			// But MGET might not preserve order if we appended missing ones.
			// For simplicity, we assume map correlation or just return list.
			// Ideally we should re-sort by CreatedAt if we mixed sources, but Timeline in Redis IS sorted.

			result := make([]models.Post, 0, len(posts))
			for _, p := range posts {
				result = append(result, *p)
			}
			return result, nil
		}
	}

	// 2. Fallback to Mongo (Aggregations) - The "Pull" Model
	// Get Friends List from Neo4j
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
	// Note: We don't strictly need to append vID here if using $or query that handles "My Posts" separately,
	// but keeping it simple.

	// 2. Build Query
	filter := bson.M{
		"$or": []bson.M{
			{
				"user_id": bson.M{"$in": friendIDs},
				"privacy": bson.M{"$in": []string{"PUBLIC", "FRIENDS"}},
				"status":  "active", // Assuming lowercase based on previous view, usually ENUM is uppercase.
			},
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

// ----------------------------- Reactions -----------------------------

func (s *FeedService) ReactToPost(ctx context.Context, userID, postID, emoji string) error {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.New("invalid post ID")
	}

	// 1. Check if post exists
	post, err := s.repo.GetPostByID(ctx, pID)
	if err != nil {
		return errors.New("post not found")
	}

	// 2. Create Reaction
	reaction := &models.Reaction{
		UserID:     uID,
		TargetID:   pID,
		TargetType: "post",
		Type:       models.ReactionType(emoji), // Casting string to ReactionType
	}

	createdReaction, err := s.repo.CreateReaction(ctx, reaction)
	if err != nil {
		return err
	}

	// 3. Increment Counter
	if err := s.repo.IncrementPostReactionCount(ctx, pID); err != nil {
		return fmt.Errorf("failed to increment reaction counter: %w", err)
	}

	// 4. Publish Event (ReactionCreated)
	reactionData, err := json.Marshal(createdReaction)
	if err == nil {
		// Whom to notify? Author of the post.
		recipientIDs := []string{post.UserID.Hex()}

		// Optional: Could notify other reactors? FB doesn't usually unless threaded.

		wsEvent := models.WebSocketEvent{
			Type:       "ReactionCreated",
			Data:       reactionData,
			Recipients: recipientIDs,
		}

		// Use "messages" topic (or "feed-events")
		if err := s.producer.PublishEvent("messages", wsEvent); err != nil {
			fmt.Printf("Failed to publish ReactionCreated: %v\n", err)
		}
	}

	return nil
}

func (s *FeedService) ReactToComment(ctx context.Context, userID, commentID, emoji string) error {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}
	cID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return errors.New("invalid comment ID")
	}

	// 1. Check if comment exists
	comment, err := s.repo.GetCommentByID(ctx, cID)
	if err != nil {
		return errors.New("comment not found")
	}

	reaction := &models.Reaction{
		UserID:     uID,
		TargetID:   cID,
		TargetType: "comment",
		Type:       models.ReactionType(emoji),
	}

	createdReaction, err := s.repo.CreateReaction(ctx, reaction)
	if err != nil {
		return err
	}

	if err := s.repo.IncrementCommentReactionCount(ctx, cID); err != nil {
		return fmt.Errorf("failed to increment reaction counter: %w", err)
	}

	reactionData, err := json.Marshal(createdReaction)
	if err == nil {
		wsEvent := models.WebSocketEvent{
			Type:       "ReactionCreated",
			Data:       reactionData,
			Recipients: []string{comment.UserID.Hex()},
		}
		s.producer.PublishEvent("messages", wsEvent)
	}
	return nil
}

func (s *FeedService) ReactToReply(ctx context.Context, userID, replyID, emoji string) error {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}
	rID, err := primitive.ObjectIDFromHex(replyID)
	if err != nil {
		return errors.New("invalid reply ID")
	}

	reply, err := s.repo.GetReplyByID(ctx, rID)
	if err != nil {
		return errors.New("reply not found")
	}

	reaction := &models.Reaction{
		UserID:     uID,
		TargetID:   rID,
		TargetType: "reply",
		Type:       models.ReactionType(emoji),
	}

	createdReaction, err := s.repo.CreateReaction(ctx, reaction)
	if err != nil {
		return err
	}

	if err := s.repo.IncrementReplyReactionCount(ctx, rID); err != nil {
		return fmt.Errorf("failed to increment reaction counter: %w", err)
	}

	reactionData, err := json.Marshal(createdReaction)
	if err == nil {
		wsEvent := models.WebSocketEvent{
			Type:       "ReactionCreated",
			Data:       reactionData,
			Recipients: []string{reply.UserID.Hex()},
		}
		s.producer.PublishEvent("messages", wsEvent)
	}
	return nil
}

// ----------------------------- Comments -----------------------------

func (s *FeedService) CreateComment(ctx context.Context, userID, postID, content string) (*models.Comment, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID")
	}

	// 1. Check if Post Exists
	post, err := s.repo.GetPostByID(ctx, pID)
	if err != nil {
		return nil, errors.New("post not found")
	}

	comment := &models.Comment{
		UserID:    uID,
		PostID:    pID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdComment, err := s.repo.CreateComment(ctx, comment)
	if err != nil {
		return nil, err
	}

	// 2. Increment Post Comment Count (Optimization: Denormalization)
	// TODO: Add IncrementPostCommentCount to repo
	// s.repo.IncrementPostCommentCount(ctx, pID)

	// 3. Publish Event
	commentData, err := json.Marshal(createdComment)
	if err == nil {
		wsEvent := models.WebSocketEvent{
			Type:       "CommentCreated",
			Data:       commentData,
			Recipients: []string{post.UserID.Hex()}, // Notify Post Author
		}
		s.producer.PublishEvent("messages", wsEvent)
	}

	return createdComment, nil
}

func (s *FeedService) ListComments(ctx context.Context, postID string, page, limit int64) ([]models.Comment, error) {
	pID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID")
	}
	filter := bson.M{"post_id": pID}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
		opts.SetSkip((page - 1) * limit)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	return s.repo.ListComments(ctx, filter, opts)
}

// ----------------------------- Replies -----------------------------

func (s *FeedService) CreateReply(ctx context.Context, userID, commentID, content string) (*models.Reply, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	cID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, errors.New("invalid comment ID")
	}

	comment, err := s.repo.GetCommentByID(ctx, cID)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	reply := &models.Reply{
		UserID:    uID,
		CommentID: cID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdReply, err := s.repo.CreateReply(ctx, reply)
	if err != nil {
		return nil, err
	}

	// Publish Event
	replyData, err := json.Marshal(createdReply)
	if err == nil {
		wsEvent := models.WebSocketEvent{
			Type: "ReplyCreated",
			Data: replyData,
			// Notify Comment Author
			Recipients: []string{comment.UserID.Hex()},
		}
		s.producer.PublishEvent("messages", wsEvent)
	}

	return createdReply, nil
}

func (s *FeedService) ListReplies(ctx context.Context, commentID string, page, limit int64) ([]models.Reply, error) {
	cID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, errors.New("invalid comment ID")
	}
	filter := bson.M{"comment_id": cID}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
		opts.SetSkip((page - 1) * limit)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: 1}}) // Replies usually ascending

	return s.repo.ListReplies(ctx, filter, opts)
}

// ----------------------------- Albums -----------------------------

func (s *FeedService) CreateAlbum(ctx context.Context, userID, title, description, privacy string) (*models.Album, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	album := &models.Album{
		UserID:      uID,
		Title:       title,
		Description: description,
		Privacy:     models.PrivacySettingType(privacy),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdAlbum, err := s.repo.CreateAlbum(ctx, album)
	if err != nil {
		return nil, err
	}

	// Publish Event
	albumData, err := json.Marshal(createdAlbum)
	if err == nil {
		// Determine recipients based on privacy (similar to Post)
		var recipientIDs []string
		if album.Privacy == "PUBLIC" || album.Privacy == "FRIENDS" {
			friends, err := s.graphRepo.GetFriendIDs(ctx, createdAlbum.UserID)
			if err == nil {
				recipientIDs = append(recipientIDs, friends...)
			}
		}
		recipientIDs = append(recipientIDs, createdAlbum.UserID.Hex())

		wsEvent := models.WebSocketEvent{
			Type:       "AlbumCreated",
			Data:       albumData,
			Recipients: recipientIDs,
		}
		s.producer.PublishEvent("messages", wsEvent)
	}

	return createdAlbum, nil
}

func (s *FeedService) GetAlbum(ctx context.Context, albumID string) (*models.Album, error) {
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return nil, errors.New("invalid album ID")
	}
	return s.repo.GetAlbumByID(ctx, aID)
}

func (s *FeedService) UpdateAlbum(ctx context.Context, albumID, title, description, privacy string) (*models.Album, error) {
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return nil, errors.New("invalid album ID")
	}

	update := bson.M{}
	if title != "" {
		update["title"] = title
	}
	if description != "" {
		update["description"] = description
	}
	if privacy != "" {
		update["privacy"] = privacy
	}

	return s.repo.UpdateAlbum(ctx, aID, update)
}

func (s *FeedService) DeleteAlbum(ctx context.Context, userID, albumID string) error {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return errors.New("invalid album ID")
	}
	return s.repo.DeleteAlbum(ctx, uID, aID)
}

func (s *FeedService) ListAlbums(ctx context.Context, userID string, page, limit int64) ([]models.Album, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
		opts.SetSkip((page - 1) * limit)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	return s.repo.ListAlbums(ctx, bson.M{"user_id": uID}, opts)
}

// ----------------------------- Album Media -----------------------------

func (s *FeedService) AddMediaToAlbum(ctx context.Context, albumID, url, mediaType, description string) (*models.AlbumMedia, error) {
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return nil, errors.New("invalid album ID")
	}

	// Verify Album Exists
	_, err = s.repo.GetAlbumByID(ctx, aID)
	if err != nil {
		return nil, errors.New("album not found")
	}

	media := &models.AlbumMedia{
		AlbumID:     aID,
		URL:         url,
		Type:        mediaType,
		Description: description,
		CreatedAt:   time.Now(),
	}

	return s.repo.AddMediaToAlbum(ctx, media)
}

func (s *FeedService) RemoveMediaFromAlbum(ctx context.Context, albumID, mediaID string) error {
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return errors.New("invalid album ID")
	}
	mID, err := primitive.ObjectIDFromHex(mediaID)
	if err != nil {
		return errors.New("invalid media ID")
	}
	return s.repo.RemoveMediaFromAlbum(ctx, aID, mID)
}

func (s *FeedService) GetAlbumMedia(ctx context.Context, albumID string, page, limit int64) ([]models.AlbumMedia, error) {
	aID, err := primitive.ObjectIDFromHex(albumID)
	if err != nil {
		return nil, errors.New("invalid album ID")
	}

	filter := bson.M{"album_id": aID}
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
		opts.SetSkip((page - 1) * limit)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	return s.repo.GetAlbumMedia(ctx, filter, opts)
}
