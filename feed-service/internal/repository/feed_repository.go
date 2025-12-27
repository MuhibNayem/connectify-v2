package repository

import (
	"context"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/events"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FeedRepository struct {
	postsCollection       *mongo.Collection
	commentsCollection    *mongo.Collection
	repliesCollection     *mongo.Collection
	reactionsCollection   *mongo.Collection
	albumsCollection      *mongo.Collection
	albumMediaCollection  *mongo.Collection
	usersCollection       *mongo.Collection // Local Replica
	friendshipsCollection *mongo.Collection // Local Replica
}

func NewFeedRepository(db *mongo.Database) *FeedRepository {
	return &FeedRepository{
		postsCollection:       db.Collection("posts"),
		commentsCollection:    db.Collection("comments"),
		repliesCollection:     db.Collection("replies"),
		reactionsCollection:   db.Collection("reactions"),
		albumsCollection:      db.Collection("albums"),
		albumMediaCollection:  db.Collection("album_media"),
		usersCollection:       db.Collection("users_replica"),
		friendshipsCollection: db.Collection("friendships_replica"),
	}
}

// ----------------------------- Posts -----------------------------

func (r *FeedRepository) CreatePost(ctx context.Context, post *models.Post) (*models.Post, error) {
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	res, err := r.postsCollection.InsertOne(ctx, post)
	if err != nil {
		return nil, err
	}
	post.ID = res.InsertedID.(primitive.ObjectID)
	return post, nil
}

func (r *FeedRepository) GetPostByID(ctx context.Context, postID primitive.ObjectID) (*models.Post, error) {
	var post models.Post
	err := r.postsCollection.FindOne(ctx, bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *FeedRepository) UpdatePost(ctx context.Context, postID primitive.ObjectID, update bson.M) (*models.Post, error) {
	update["updated_at"] = time.Now()
	res := r.postsCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": postID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updatedPost models.Post
	if err := res.Decode(&updatedPost); err != nil {
		return nil, err
	}
	return &updatedPost, nil
}

func (r *FeedRepository) UpdatePostStatus(ctx context.Context, postID primitive.ObjectID, status string) error {
	_, err := r.postsCollection.UpdateOne(ctx, bson.M{"_id": postID}, bson.M{
		"$set": bson.M{"status": status, "updated_at": time.Now()},
	})
	return err
}

func (r *FeedRepository) DeletePost(ctx context.Context, userID, postID primitive.ObjectID) error {
	res, err := r.postsCollection.DeleteOne(ctx, bson.M{"_id": postID, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ... (Other methods will be ported incrementally or genericized) ...
// For the MVP, we need ListPosts and basic CRUD.

func (r *FeedRepository) ListPosts(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Post, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Aggregation pipeline to join Authors, etc.
	// NOTE: This assumes 'users' collection is in the same DB, which fits our "Shared DB" plan.
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		// Lookup User (Author)
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users_replica",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "author_info",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$author_info", "preserveNullAndEmptyArrays": true}}},
		// Project Author fields
		bson.D{{Key: "$addFields", Value: bson.M{
			"author": bson.M{
				"id":        bson.M{"$toString": "$author_info._id"},
				"username":  "$author_info.username",
				"full_name": "$author_info.full_name",
				"avatar":    "$author_info.avatar",
			},
		}}},
	}

	if opts != nil {
		if opts.Sort != nil {
			pipeline = append(pipeline, bson.D{{Key: "$sort", Value: opts.Sort}})
		}
		if opts.Skip != nil {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: *opts.Skip}})
		}
		if opts.Limit != nil {
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: *opts.Limit}})
		}
	}

	cur, err := r.postsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var posts []models.Post
	if err := cur.All(ctx, &posts); err != nil {
		return nil, err
	}
	if posts == nil {
		posts = []models.Post{}
	}
	return posts, nil
}

func (r *FeedRepository) CountPosts(ctx context.Context, filter bson.M) (int64, error) {
	return r.postsCollection.CountDocuments(ctx, filter)
}

func (r *FeedRepository) GetPostsByHashtag(ctx context.Context, hashtag string, limit, offset int64) ([]models.Post, error) {
	filter := bson.M{
		"hashtags": hashtag, // Simple array match
		"status":   "ACTIVE",
		"privacy":  "PUBLIC", // Hashtag searches are public usually
	}
	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.postsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []models.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// ----------------------------- Reactions -----------------------------

func (r *FeedRepository) CreateReaction(ctx context.Context, reaction *models.Reaction) (*models.Reaction, error) {
	reaction.CreatedAt = time.Now()
	// Upsert to prevent duplicate reactions from same user
	filter := bson.M{
		"user_id":     reaction.UserID,
		"target_id":   reaction.TargetID,
		"target_type": reaction.TargetType,
	}
	update := bson.M{
		"$set": reaction,
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.reactionsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, err
	}
	// We might need to fetch the inserted ID if new, but for reaction toggling this is often sufficient.
	// However, to return a complete model with ID:
	var savedReaction models.Reaction
	err = r.reactionsCollection.FindOne(ctx, filter).Decode(&savedReaction)
	if err != nil {
		return nil, err
	}
	return &savedReaction, nil
}

func (r *FeedRepository) DeleteReaction(ctx context.Context, userID, targetID primitive.ObjectID, targetType string) error {
	filter := bson.M{
		"user_id":     userID,
		"target_id":   targetID,
		"target_type": targetType,
	}
	_, err := r.reactionsCollection.DeleteOne(ctx, filter)
	return err
}

func (r *FeedRepository) ListReactions(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Reaction, error) {
	cursor, err := r.reactionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reactions []models.Reaction
	if err := cursor.All(ctx, &reactions); err != nil {
		return nil, err
	}
	return reactions, nil
}

// Post Reaction Counters
func (r *FeedRepository) IncrementPostReactionCount(ctx context.Context, postID primitive.ObjectID) error {
	_, err := r.postsCollection.UpdateOne(ctx, bson.M{"_id": postID}, bson.M{"$inc": bson.M{"total_reactions": 1}})
	return err
}

func (r *FeedRepository) DecrementPostReactionCount(ctx context.Context, postID primitive.ObjectID) error {
	_, err := r.postsCollection.UpdateOne(ctx, bson.M{"_id": postID}, bson.M{"$inc": bson.M{"total_reactions": -1}})
	return err
}

func (r *FeedRepository) IncrementCommentReactionCount(ctx context.Context, commentID primitive.ObjectID) error {
	_, err := r.commentsCollection.UpdateOne(ctx, bson.M{"_id": commentID}, bson.M{"$inc": bson.M{"total_reactions": 1}})
	return err
}

func (r *FeedRepository) DecrementCommentReactionCount(ctx context.Context, commentID primitive.ObjectID) error {
	_, err := r.commentsCollection.UpdateOne(ctx, bson.M{"_id": commentID}, bson.M{"$inc": bson.M{"total_reactions": -1}})
	return err
}

func (r *FeedRepository) IncrementReplyReactionCount(ctx context.Context, replyID primitive.ObjectID) error {
	_, err := r.repliesCollection.UpdateOne(ctx, bson.M{"_id": replyID}, bson.M{"$inc": bson.M{"total_reactions": 1}})
	return err
}

func (r *FeedRepository) DecrementReplyReactionCount(ctx context.Context, replyID primitive.ObjectID) error {
	_, err := r.repliesCollection.UpdateOne(ctx, bson.M{"_id": replyID}, bson.M{"$inc": bson.M{"total_reactions": -1}})
	return err
}

func (r *FeedRepository) GetFriends(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	// Find all accepted friendships where userID is requester OR receiver
	filter := bson.M{
		"status": "accepted",
		"$or": []bson.M{
			{"requester_id": userID},
			{"receiver_id": userID},
		},
	}

	cursor, err := r.friendshipsCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friendships []struct {
		RequesterID primitive.ObjectID `bson:"requester_id"`
		ReceiverID  primitive.ObjectID `bson:"receiver_id"`
	}

	if err := cursor.All(ctx, &friendships); err != nil {
		return nil, err
	}

	var friendIDs []primitive.ObjectID
	for _, f := range friendships {
		if f.RequesterID == userID {
			friendIDs = append(friendIDs, f.ReceiverID)
		} else {
			friendIDs = append(friendIDs, f.RequesterID)
		}
	}
	return friendIDs, nil
}

// ----------------------------- Data Replication -----------------------------

func (r *FeedRepository) UpsertUserReplica(ctx context.Context, event *events.UserUpdatedEvent) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": event.UserID} // Storing as string or ObjectID depending on event content
	// Convert ID if needed, assuming Hex string in event
	oid, _ := primitive.ObjectIDFromHex(event.UserID)
	filter = bson.M{"_id": oid}

	update := bson.M{
		"$set": bson.M{
			"username":      event.Username,
			"full_name":     event.FullName,
			"avatar":        event.Avatar,
			"date_of_birth": event.DateOfBirth,
			"updated_at":    time.Now(),
		},
	}
	_, err := r.usersCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *FeedRepository) UpdateFriendshipReplica(ctx context.Context, event *events.FriendshipEvent) error {
	// We only care about Accepted/Removed friendships for feed visibility
	// Can also store "Blocked" status
	// Upsert or Delete based on Status

	requesterOID, _ := primitive.ObjectIDFromHex(event.RequesterID)
	receiverOID, _ := primitive.ObjectIDFromHex(event.ReceiverID)

	filter := bson.M{
		"requester_id": requesterOID,
		"receiver_id":  receiverOID,
	}

	opts := options.Update().SetUpsert(true)

	if event.Status == "accepted" {
		update := bson.M{
			"$set": bson.M{
				"status":     "accepted",
				"created_at": event.Timestamp,
				"updated_at": time.Now(),
			},
		}
		_, err := r.friendshipsCollection.UpdateOne(ctx, filter, update, opts)
		return err
	} else if event.Status == "removed" || event.Status == "blocked" {
		// Soft delete or Hard delete? Hard delete for simplicity in building graph query
		_, err := r.friendshipsCollection.DeleteOne(ctx, filter)
		return err
	}
	return nil
}

func (r *FeedRepository) GetCommentByID(ctx context.Context, commentID primitive.ObjectID) (*models.Comment, error) {
	var comment models.Comment
	err := r.commentsCollection.FindOne(ctx, bson.M{"_id": commentID}).Decode(&comment)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *FeedRepository) GetReplyByID(ctx context.Context, replyID primitive.ObjectID) (*models.Reply, error) {
	var reply models.Reply
	err := r.repliesCollection.FindOne(ctx, bson.M{"_id": replyID}).Decode(&reply)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

// ----------------------------- Comments -----------------------------

func (r *FeedRepository) CreateComment(ctx context.Context, comment *models.Comment) (*models.Comment, error) {
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	res, err := r.commentsCollection.InsertOne(ctx, comment)
	if err != nil {
		return nil, err
	}
	comment.ID = res.InsertedID.(primitive.ObjectID)
	return comment, nil
}

func (r *FeedRepository) UpdateComment(ctx context.Context, commentID primitive.ObjectID, update bson.M) (*models.Comment, error) {
	update["updated_at"] = time.Now()
	res := r.commentsCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": commentID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updatedComment models.Comment
	if err := res.Decode(&updatedComment); err != nil {
		return nil, err
	}
	return &updatedComment, nil
}

func (r *FeedRepository) DeleteComment(ctx context.Context, postID, commentID primitive.ObjectID) error {
	res, err := r.commentsCollection.DeleteOne(ctx, bson.M{"_id": commentID, "post_id": postID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *FeedRepository) ListComments(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Comment, error) {
	// Lookup Author info similar to Posts
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users_replica",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "author_info",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$author_info", "preserveNullAndEmptyArrays": true}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"author": bson.M{
				"id":        bson.M{"$toString": "$author_info._id"},
				"username":  "$author_info.username",
				"full_name": "$author_info.full_name",
				"avatar":    "$author_info.avatar",
			},
		}}},
	}

	if opts != nil {
		if opts.Sort != nil {
			pipeline = append(pipeline, bson.D{{Key: "$sort", Value: opts.Sort}})
		}
		if opts.Skip != nil {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: *opts.Skip}})
		}
		if opts.Limit != nil {
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: *opts.Limit}})
		}
	}

	cur, err := r.commentsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var comments []models.Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []models.Comment{}
	}
	return comments, nil
}

// ----------------------------- Replies -----------------------------

func (r *FeedRepository) CreateReply(ctx context.Context, reply *models.Reply) (*models.Reply, error) {
	reply.CreatedAt = time.Now()
	reply.UpdatedAt = time.Now()
	res, err := r.repliesCollection.InsertOne(ctx, reply)
	if err != nil {
		return nil, err
	}
	reply.ID = res.InsertedID.(primitive.ObjectID)
	return reply, nil
}

func (r *FeedRepository) UpdateReply(ctx context.Context, replyID primitive.ObjectID, update bson.M) (*models.Reply, error) {
	update["updated_at"] = time.Now()
	res := r.repliesCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": replyID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updatedReply models.Reply
	if err := res.Decode(&updatedReply); err != nil {
		return nil, err
	}
	return &updatedReply, nil
}

func (r *FeedRepository) DeleteReply(ctx context.Context, commentID, replyID primitive.ObjectID) error {
	res, err := r.repliesCollection.DeleteOne(ctx, bson.M{"_id": replyID, "comment_id": commentID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *FeedRepository) ListReplies(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Reply, error) {
	// Lookup Author info
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users_replica",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "author_info",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$author_info", "preserveNullAndEmptyArrays": true}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"author": bson.M{
				"id":        bson.M{"$toString": "$author_info._id"},
				"username":  "$author_info.username",
				"full_name": "$author_info.full_name",
				"avatar":    "$author_info.avatar",
			},
		}}},
	}

	if opts != nil {
		if opts.Sort != nil {
			pipeline = append(pipeline, bson.D{{Key: "$sort", Value: opts.Sort}})
		}
		if opts.Skip != nil {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: *opts.Skip}})
		}
		if opts.Limit != nil {
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: *opts.Limit}})
		}
	}

	cur, err := r.repliesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var replies []models.Reply
	if err := cur.All(ctx, &replies); err != nil {
		return nil, err
	}
	if replies == nil {
		replies = []models.Reply{}
	}
	return replies, nil
}

// ----------------------------- Albums -----------------------------

func (r *FeedRepository) CreateAlbum(ctx context.Context, album *models.Album) (*models.Album, error) {
	album.CreatedAt = time.Now()
	album.UpdatedAt = time.Now()
	res, err := r.albumsCollection.InsertOne(ctx, album)
	if err != nil {
		return nil, err
	}
	album.ID = res.InsertedID.(primitive.ObjectID)
	return album, nil
}

func (r *FeedRepository) GetAlbumByID(ctx context.Context, albumID primitive.ObjectID) (*models.Album, error) {
	var album models.Album
	err := r.albumsCollection.FindOne(ctx, bson.M{"_id": albumID}).Decode(&album)
	if err != nil {
		return nil, err
	}
	return &album, nil
}

func (r *FeedRepository) UpdateAlbum(ctx context.Context, albumID primitive.ObjectID, update bson.M) (*models.Album, error) {
	update["updated_at"] = time.Now()
	res := r.albumsCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": albumID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updatedAlbum models.Album
	if err := res.Decode(&updatedAlbum); err != nil {
		return nil, err
	}
	return &updatedAlbum, nil
}

func (r *FeedRepository) DeleteAlbum(ctx context.Context, userID, albumID primitive.ObjectID) error {
	res, err := r.albumsCollection.DeleteOne(ctx, bson.M{"_id": albumID, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *FeedRepository) ListAlbums(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Album, error) {
	cur, err := r.albumsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var albums []models.Album
	if err := cur.All(ctx, &albums); err != nil {
		return nil, err
	}
	if albums == nil {
		albums = []models.Album{}
	}
	return albums, nil
}

// ----------------------------- Album Media -----------------------------

func (r *FeedRepository) AddMediaToAlbum(ctx context.Context, media *models.AlbumMedia) (*models.AlbumMedia, error) {
	media.CreatedAt = time.Now()
	res, err := r.albumMediaCollection.InsertOne(ctx, media)
	if err != nil {
		return nil, err
	}
	media.ID = res.InsertedID.(primitive.ObjectID)
	return media, nil
}

func (r *FeedRepository) RemoveMediaFromAlbum(ctx context.Context, albumID, mediaID primitive.ObjectID) error {
	res, err := r.albumMediaCollection.DeleteOne(ctx, bson.M{"_id": mediaID, "album_id": albumID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *FeedRepository) GetAlbumMedia(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.AlbumMedia, error) {
	cur, err := r.albumMediaCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var mediaList []models.AlbumMedia
	if err := cur.All(ctx, &mediaList); err != nil {
		return nil, err
	}
	if mediaList == nil {
		mediaList = []models.AlbumMedia{}
	}
	return mediaList, nil
}
