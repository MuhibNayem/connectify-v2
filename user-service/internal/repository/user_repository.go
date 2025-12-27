package repository

import (
	"context"
	"log"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	// Create indexes
	_, err := db.Collection("users").Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		log.Printf("Failed to create user indexes: %v", err)
	}

	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	result, err := r.db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return user, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var user models.User
	err := r.db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindUserByUserName(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var user models.User
	err := r.db.Collection("users").FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var user models.User
	err := r.db.Collection("users").FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	update["updated_at"] = time.Now()

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := r.db.Collection("users").FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
		opts,
	)

	var updatedUser models.User
	if err := result.Decode(&updatedUser); err != nil {
		return nil, err
	}

	return &updatedUser, nil
}

// AddFriend adds friend to mongo array (Legacy/Redundant but kept for read compatibility if needed)
func (r *UserRepository) AddFriend(ctx context.Context, userID1, userID2 primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	session, err := r.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err = r.db.Collection("users").UpdateOne(sessCtx, bson.M{"_id": userID1}, bson.M{"$addToSet": bson.M{"friends": userID2}})
		if err != nil {
			return nil, err
		}
		_, err = r.db.Collection("users").UpdateOne(sessCtx, bson.M{"_id": userID2}, bson.M{"$addToSet": bson.M{"friends": userID1}})
		return nil, err
	})
	return err
}

func (r *UserRepository) RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := r.db.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$pull": bson.M{"friends": friendID}})
	return err
}

func (r *UserRepository) CountUsers(ctx context.Context, filter bson.M) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return r.db.Collection("users").CountDocuments(ctx, filter)
}

func (r *UserRepository) FindUsers(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursor, err := r.db.Collection("users").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// FindUsersByIDs fetches multiple users by IDs using $in operator (batch fetch)
func (r *UserRepository) FindUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if len(ids) == 0 {
		return []models.User{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := r.db.Collection("users").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
