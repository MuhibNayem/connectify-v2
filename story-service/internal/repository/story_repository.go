package repository

import (
	"context"
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StoryRepository struct {
	collection          *mongo.Collection
	viewsCollection     *mongo.Collection
	reactionsCollection *mongo.Collection
}

func NewStoryRepository(db *mongo.Database) *StoryRepository {
	// Create indexes for stories
	_, err := db.Collection("stories").Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index()},
			{Keys: bson.D{{Key: "created_at", Value: -1}}, Options: options.Index()},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index()},
		},
	)
	if err != nil {
		panic("Failed to create story indexes: " + err.Error())
	}

	// Create indexes for story_views
	_, err = db.Collection("story_views").Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{Keys: bson.D{{Key: "story_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "story_id", Value: 1}}, Options: options.Index()},
		},
	)
	if err != nil {
		panic("Failed to create story_views indexes: " + err.Error())
	}

	// Create indexes for story_reactions
	_, err = db.Collection("story_reactions").Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{Keys: bson.D{{Key: "story_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index()},
			{Keys: bson.D{{Key: "story_id", Value: 1}}, Options: options.Index()},
		},
	)
	if err != nil {
		panic("Failed to create story_reactions indexes: " + err.Error())
	}

	return &StoryRepository{
		collection:          db.Collection("stories"),
		viewsCollection:     db.Collection("story_views"),
		reactionsCollection: db.Collection("story_reactions"),
	}
}

func (r *StoryRepository) CreateStory(ctx context.Context, story *models.Story) (*models.Story, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	story.CreatedAt = time.Now()
	if story.ExpiresAt.IsZero() {
		story.ExpiresAt = story.CreatedAt.Add(24 * time.Hour)
	}

	res, err := r.collection.InsertOne(ctx, story)
	if err != nil {
		return nil, err
	}
	story.ID = res.InsertedID.(primitive.ObjectID)
	return story, nil
}

func (r *StoryRepository) GetStoryByID(ctx context.Context, id primitive.ObjectID) (*models.Story, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var story models.Story
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&story)
	if err != nil {
		return nil, err
	}
	return &story, nil
}

func (r *StoryRepository) DeleteStory(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Delete associated views
	_, _ = r.viewsCollection.DeleteMany(ctx, bson.M{"story_id": id})

	// Delete associated reactions
	_, _ = r.reactionsCollection.DeleteMany(ctx, bson.M{"story_id": id})

	// Delete the story
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	return err
}

func (r *StoryRepository) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now()
	filter := bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$gt": now},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var stories []models.Story
	if err := cur.All(ctx, &stories); err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) GetActiveStoryAuthors(ctx context.Context, viewerID primitive.ObjectID, userIDs []primitive.ObjectID, limit, offset int) ([]primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now()

	// Privacy Filter
	privacyMatch := bson.M{
		"$or": []bson.M{
			{"user_id": viewerID},
			{
				"user_id": bson.M{"$ne": viewerID},
				"$or": []bson.M{
					{"privacy": bson.M{"$in": []string{string(models.PrivacySettingPublic), string(models.PrivacySettingFriends)}}},
					{"privacy": models.PrivacySettingCustom, "allowed_viewers": viewerID},
					{"privacy": models.PrivacySettingFriendsExcept, "blocked_viewers": bson.M{"$ne": viewerID}},
				},
			},
		},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"$and": []bson.M{
				{"user_id": bson.M{"$in": userIDs}},
				{"expires_at": bson.M{"$gt": now}},
				privacyMatch,
			},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$user_id"},
			{Key: "latest_story", Value: bson.D{{Key: "$max", Value: "$created_at"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "latest_story", Value: -1}}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},
	}

	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}

	authors := make([]primitive.ObjectID, len(results))
	for i, res := range results {
		authors[i] = res.ID
	}
	return authors, nil
}

func (r *StoryRepository) GetStoriesForUsers(ctx context.Context, viewerID primitive.ObjectID, authorIDs []primitive.ObjectID) ([]models.Story, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now()

	privacyMatch := bson.M{
		"$or": []bson.M{
			{"user_id": viewerID},
			{
				"user_id": bson.M{"$ne": viewerID},
				"$or": []bson.M{
					{"privacy": bson.M{"$in": []string{string(models.PrivacySettingPublic), string(models.PrivacySettingFriends)}}},
					{"privacy": models.PrivacySettingCustom, "allowed_viewers": viewerID},
					{"privacy": models.PrivacySettingFriendsExcept, "blocked_viewers": bson.M{"$ne": viewerID}},
				},
			},
		},
	}

	filter := bson.M{
		"$and": []bson.M{
			{"user_id": bson.M{"$in": authorIDs}},
			{"expires_at": bson.M{"$gt": now}},
			privacyMatch,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var stories []models.Story
	if err := cur.All(ctx, &stories); err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) AddViewer(ctx context.Context, storyID primitive.ObjectID, viewerID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	view := models.StoryView{
		ID:       primitive.NewObjectID(),
		StoryID:  storyID,
		UserID:   viewerID,
		ViewedAt: time.Now(),
	}
	_, err := r.viewsCollection.InsertOne(ctx, view)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil // Already viewed
		}
		return err
	}

	// Increment view count
	filter := bson.M{"_id": storyID}
	update := bson.M{"$inc": bson.M{"view_count": 1}}
	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *StoryRepository) AddReaction(ctx context.Context, storyID primitive.ObjectID, reaction models.StoryReaction) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reaction.ID = primitive.NewObjectID()
	_, err := r.reactionsCollection.InsertOne(ctx, reaction)
	if err != nil {
		return err
	}

	// Increment reaction count
	filter := bson.M{"_id": storyID}
	update := bson.M{"$inc": bson.M{"reaction_count": 1}}
	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *StoryRepository) GetStoryViewersWithReactions(ctx context.Context, storyID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"story_id": storyID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$lookup", Value: bson.M{
			"from": "story_reactions",
			"let":  bson.M{"uid": "$user_id", "sid": "$story_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": []bson.M{
							{"$eq": []interface{}{"$story_id", "$$sid"}},
							{"$eq": []interface{}{"$user_id", "$$uid"}},
						},
					},
				}}},
				{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				{{Key: "$limit", Value: 1}},
			},
			"as": "user_reaction",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$user_reaction",
			"preserveNullAndEmptyArrays": true,
		}}},
		{{Key: "$sort", Value: bson.M{"viewed_at": -1}}},
		{{Key: "$project", Value: bson.M{
			"_id": 0,
			"user": bson.M{
				"_id":       "$user._id",
				"username":  "$user.username",
				"full_name": "$user.full_name",
				"avatar":    "$user.avatar",
			},
			"reaction_type": "$user_reaction.type",
			"viewed_at":     "$viewed_at",
		}}},
	}

	cur, err := r.viewsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []models.StoryViewerResponse
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetExpiredStories returns stories that have expired
func (r *StoryRepository) GetExpiredStories(ctx context.Context) ([]models.Story, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	now := time.Now()
	filter := bson.M{"expires_at": bson.M{"$lte": now}}
	opts := options.Find().SetLimit(100)

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var stories []models.Story
	if err := cur.All(ctx, &stories); err != nil {
		return nil, err
	}
	return stories, nil
}

// DeleteStories bulk deletes stories by their IDs
func (r *StoryRepository) DeleteStories(ctx context.Context, ids []primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Delete views
	_, _ = r.viewsCollection.DeleteMany(ctx, bson.M{"story_id": bson.M{"$in": ids}})

	// Delete reactions
	_, _ = r.reactionsCollection.DeleteMany(ctx, bson.M{"story_id": bson.M{"$in": ids}})

	// Delete stories
	_, err := r.collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}
