package integration

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserLocalRepository manages the local replica of user data for the events domain.
type UserLocalRepository struct {
	collection *mongo.Collection
}

func NewUserLocalRepository(db *mongo.Database) *UserLocalRepository {
	return &UserLocalRepository{
		collection: db.Collection("replicated_users"),
	}
}

func (r *UserLocalRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*EventUser, error) {
	var user EventUser
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserLocalRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]EventUser, error) {
	var users []EventUser
	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserLocalRepository) UpsertUser(ctx context.Context, user *EventUser) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user}
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *UserLocalRepository) FindFriendBirthdays(ctx context.Context, friendIDs []primitive.ObjectID) ([]EventUser, []EventUser, error) {
	now := time.Now()
	currentMonth := int(now.Month())
	currentDay := now.Day()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$in", Value: friendIDs}}},
			{Key: "date_of_birth", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: nil}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "username", Value: 1},
			{Key: "full_name", Value: 1},
			{Key: "avatar", Value: 1},
			{Key: "date_of_birth", Value: 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var friends []EventUser
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, nil, err
	}

	var today []EventUser
	var upcoming []EventUser

	for _, f := range friends {
		dob := f.DateOfBirth
		if dob == nil {
			continue
		}

		dMonth := int(dob.Month())
		dDay := dob.Day()

		isToday := dMonth == currentMonth && dDay == currentDay
		if isToday {
			today = append(today, f)
			continue
		}

		thisYearBday := time.Date(now.Year(), dob.Month(), dob.Day(), 0, 0, 0, 0, now.Location())
		if thisYearBday.Before(now) {
			thisYearBday = thisYearBday.AddDate(1, 0, 0)
		}

		daysUntil := int(thisYearBday.Sub(now).Hours() / 24)
		if daysUntil >= 0 && daysUntil <= 30 {
			upcoming = append(upcoming, f)
		}
	}

	return today, upcoming, nil
}

func (r *UserLocalRepository) AddFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$addToSet": bson.M{"friends": friendID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *UserLocalRepository) RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$pull": bson.M{"friends": friendID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *UserLocalRepository) GetFriends(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	var user EventUser
	opts := options.FindOne().SetProjection(bson.M{"friends": 1})
	err := r.collection.FindOne(ctx, bson.M{"_id": userID}, opts).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []primitive.ObjectID{}, nil
		}
		return nil, err
	}
	return user.Friends, nil
}
