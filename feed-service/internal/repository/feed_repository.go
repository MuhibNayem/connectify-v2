package repository

import (
	"context"
	"time"

	"gitlab.com/spydotech-group/shared-entity/events"
	"gitlab.com/spydotech-group/shared-entity/models"
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
