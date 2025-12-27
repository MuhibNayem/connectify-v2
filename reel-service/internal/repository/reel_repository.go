package repository

import (
	"context"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReelRepository struct {
	collection          *mongo.Collection
	commentsCollection  *mongo.Collection
	reactionsCollection *mongo.Collection
}

func NewReelRepository(db *mongo.Database) *ReelRepository {
	ctx := context.Background()

	db.Collection("reels").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "views", Value: -1}}},
	})

	db.Collection("reel_comments").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "reel_id", Value: 1}},
	})

	return &ReelRepository{
		collection:          db.Collection("reels"),
		commentsCollection:  db.Collection("reel_comments"),
		reactionsCollection: db.Collection("reel_reactions"),
	}
}

func (r *ReelRepository) CreateReel(ctx context.Context, reel *models.Reel) (*models.Reel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reel.CreatedAt = time.Now()
	reel.UpdatedAt = time.Now()

	res, err := r.collection.InsertOne(ctx, reel)
	if err != nil {
		return nil, err
	}
	reel.ID = res.InsertedID.(primitive.ObjectID)
	return reel, nil
}

func (r *ReelRepository) GetReelByID(ctx context.Context, id primitive.ObjectID) (*models.Reel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var reel models.Reel
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&reel)
	if err != nil {
		return nil, err
	}
	return &reel, nil
}

func (r *ReelRepository) GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var reels []models.Reel
	if err := cur.All(ctx, &reels); err != nil {
		return nil, err
	}
	return reels, nil
}

func (r *ReelRepository) DeleteReel(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	return err
}

func (r *ReelRepository) IncrementViews(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"views": 1}})
	return err
}

func (r *ReelRepository) GetReelsFeed(ctx context.Context, userID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int64) ([]models.Reel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"privacy": "PUBLIC"},
			{"user_id": userID},
			{
				"privacy": "FRIENDS",
				"user_id": bson.M{"$in": friendIDs},
			},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var reels []models.Reel
	if err := cur.All(ctx, &reels); err != nil {
		return nil, err
	}
	return reels, nil
}

func (r *ReelRepository) AddComment(ctx context.Context, reelID primitive.ObjectID, comment models.Comment) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	comment.ReelID = &reelID
	comment.CreatedAt = time.Now()

	_, err := r.commentsCollection.InsertOne(ctx, comment)
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": reelID}, bson.M{"$inc": bson.M{"comments": 1}})
	return err
}

func (r *ReelRepository) GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cur, err := r.commentsCollection.Find(ctx, bson.M{"reel_id": reelID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var comments []models.Comment
	if err := cur.All(ctx, &comments); err != nil {
		if err == mongo.ErrNoDocuments {
			return []models.Comment{}, nil
		}
		return nil, err
	}
	return comments, nil
}

func (r *ReelRepository) AddReply(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, reply models.Reply) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.commentsCollection.UpdateOne(ctx, bson.M{"_id": commentID}, bson.M{"$push": bson.M{"replies": reply}})
	return err
}

func (r *ReelRepository) GetReaction(ctx context.Context, targetID primitive.ObjectID, userID primitive.ObjectID) (*models.Reaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var reaction models.Reaction
	err := r.reactionsCollection.FindOne(ctx, bson.M{"target_id": targetID, "user_id": userID}).Decode(&reaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &reaction, nil
}

func (r *ReelRepository) AddReaction(ctx context.Context, reaction *models.Reaction) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.reactionsCollection.InsertOne(ctx, reaction)
	if err != nil {
		return err
	}

	incMap := bson.M{"reaction_counts." + string(reaction.Type): 1}
	if reaction.Type == "LIKE" {
		incMap["likes"] = 1
	}

	if reaction.TargetType == "reel" {
		_, err = r.collection.UpdateOne(ctx, bson.M{"_id": reaction.TargetID}, bson.M{"$inc": incMap})
	}
	return err
}

func (r *ReelRepository) RemoveReaction(ctx context.Context, reaction *models.Reaction) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.reactionsCollection.DeleteOne(ctx, bson.M{"target_id": reaction.TargetID, "user_id": reaction.UserID})
	if err != nil {
		return err
	}

	decMap := bson.M{"reaction_counts." + string(reaction.Type): -1}
	if reaction.Type == "LIKE" {
		decMap["likes"] = -1
	}

	if reaction.TargetType == "reel" {
		_, err = r.collection.UpdateOne(ctx, bson.M{"_id": reaction.TargetID}, bson.M{"$inc": decMap})
	}
	return err
}

func (r *ReelRepository) ReactToComment(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, userID primitive.ObjectID, reactionType models.ReactionType) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	r.commentsCollection.UpdateOne(ctx, bson.M{"_id": commentID}, bson.M{"$pull": bson.M{"reactions": bson.M{"user_id": userID}}})

	reaction := models.Reaction{
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		TargetID:   commentID,
		TargetType: "comment",
		Type:       reactionType,
		CreatedAt:  time.Now(),
	}

	_, err := r.commentsCollection.UpdateOne(ctx, bson.M{"_id": commentID}, bson.M{"$push": bson.M{"reactions": reaction}})
	return err
}

// ListReels returns all reels with pagination, sorted by creation date (descending)
func (r *ReelRepository) ListReels(ctx context.Context, limit int64, offset int64) ([]models.Reel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit).SetSkip(offset)

	cur, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var reels []models.Reel
	if err := cur.All(ctx, &reels); err != nil {
		return nil, err
	}
	return reels, nil
}

// UpdateAuthorInfo updates the author info in all reels by this user
func (r *ReelRepository) UpdateAuthorInfo(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.collection.UpdateMany(
		ctx,
		bson.M{"user_id": userID},
		bson.M{
			"$set": bson.M{
				"author":     author,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}
