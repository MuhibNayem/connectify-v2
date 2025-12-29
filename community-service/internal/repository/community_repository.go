package repository

import (
	"context"
	"errors"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommunityRepository struct {
	collection *mongo.Collection
}

func NewCommunityRepository(db *mongo.Database) *CommunityRepository {
	return &CommunityRepository{
		collection: db.Collection("communities"),
	}
}

func (r *CommunityRepository) Create(ctx context.Context, community *models.Community) error {
	community.CreatedAt = time.Now()
	community.UpdatedAt = time.Now()
	// Stats initialized by service or here? Original repo did here.
	if community.Stats.MemberCount == 0 && len(community.Members) > 0 {
		community.Stats.MemberCount = int64(len(community.Members))
	}
	// Note: Existing code set PostCount to 0 hardcoded.

	result, err := r.collection.InsertOne(ctx, community)
	if err != nil { // Check for duplicates
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("community with this name or slug already exists")
		}
		return err
	}

	community.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *CommunityRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Community, error) {
	var community models.Community
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&community)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// return standard error or nil? Service expects error according to old code
			return nil, errors.New("community not found")
		}
		return nil, err
	}
	return &community, nil
}

func (r *CommunityRepository) Update(ctx context.Context, community *models.Community) error {
	community.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": community.ID}, community)
	return err
}

func (r *CommunityRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List returns communities based on filters.
// Note: Original had logic for Public/Visible.
func (r *CommunityRepository) List(ctx context.Context, limit, page int64) ([]models.Community, int64, error) {
	skip := (page - 1) * limit
	opts := options.Find().SetLimit(limit).SetSkip(skip).SetSort(bson.M{"created_at": -1})

	filter := bson.M{
		"$or": []bson.M{
			{"privacy": models.CommunityPrivacyPublic},
			{"visibility": models.CommunityVisibilityVisible},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var communities []models.Community
	if err = cursor.All(ctx, &communities); err != nil {
		return nil, 0, err
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return communities, total, nil
}

func (r *CommunityRepository) AddMember(ctx context.Context, communityID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": communityID}
	update := bson.M{
		"$addToSet": bson.M{"members": userID},
		"$inc":      bson.M{"stats.member_count": 1},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CommunityRepository) RemoveMember(ctx context.Context, communityID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": communityID}
	update := bson.M{
		"$pull": bson.M{"members": userID, "admins": userID},
		"$inc":  bson.M{"stats.member_count": -1},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CommunityRepository) AddPendingMember(ctx context.Context, communityID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": communityID}
	update := bson.M{"$addToSet": bson.M{"pending_members": userID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CommunityRepository) RemovePendingMember(ctx context.Context, communityID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": communityID}
	update := bson.M{"$pull": bson.M{"pending_members": userID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CommunityRepository) Search(ctx context.Context, query string, limit, page int64) ([]models.Community, int64, error) {
	skip := (page - 1) * limit
	opts := options.Find().SetLimit(limit).SetSkip(skip)

	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"name": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
					{"description": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
				},
			},
			{"visibility": bson.M{"$ne": models.CommunityVisibilityHidden}},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var communities []models.Community
	if err = cursor.All(ctx, &communities); err != nil {
		return nil, 0, err
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return communities, total, nil
}

func (r *CommunityRepository) GetUserCommunities(ctx context.Context, userID primitive.ObjectID) ([]models.Community, error) {
	filter := bson.M{"members": userID}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var communities []models.Community
	if err = cursor.All(ctx, &communities); err != nil {
		return nil, err
	}
	return communities, nil
}

// GetMembers performs lookup on "users" collection. Assumes Shared Database.
func (r *CommunityRepository) GetMembers(ctx context.Context, communityID primitive.ObjectID, limit, page int64) ([]models.User, int64, error) {
	skip := (page - 1) * limit

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: communityID}}}},
		{{Key: "$project", Value: bson.D{{Key: "members", Value: 1}, {Key: "_id", Value: 0}}}},
		{{Key: "$unwind", Value: "$members"}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "members"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user"},
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$user"}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	// Count logic
	var community models.Community
	err = r.collection.FindOne(ctx, bson.M{"_id": communityID}).Decode(&community)
	if err != nil {
		return nil, 0, err
	}

	if users == nil {
		users = []models.User{}
	}
	return users, community.Stats.MemberCount, nil
}

func (r *CommunityRepository) GetAdmins(ctx context.Context, communityID primitive.ObjectID) ([]models.User, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: communityID}}}},
		{{Key: "$project", Value: bson.D{{Key: "admins", Value: 1}, {Key: "_id", Value: 0}}}},
		{{Key: "$unwind", Value: "$admins"}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "admins"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user"},
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$user"}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

func (r *CommunityRepository) GetPendingMembers(ctx context.Context, communityID primitive.ObjectID, limit, page int64) ([]models.User, int64, error) {
	skip := (page - 1) * limit

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: communityID}}}},
		{{Key: "$project", Value: bson.D{{Key: "pending_members", Value: 1}, {Key: "_id", Value: 0}}}},
		{{Key: "$unwind", Value: "$pending_members"}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "pending_members"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user"},
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$user"}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	// Count pending members
	var community models.Community
	err = r.collection.FindOne(ctx, bson.M{"_id": communityID}).Decode(&community)
	if err != nil {
		return nil, 0, err
	}

	if users == nil {
		users = []models.User{}
	}
	return users, int64(len(community.PendingMembers)), nil
}
