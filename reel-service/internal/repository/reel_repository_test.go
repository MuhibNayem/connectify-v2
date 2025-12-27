package repository

import (
	"context"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreateReel(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection:          mt.Coll,
			commentsCollection:  mt.Coll,
			reactionsCollection: mt.Coll,
		}

		userID := primitive.NewObjectID()
		reel := &models.Reel{
			UserID:   userID,
			VideoURL: "https://example.com/video.mp4",
			Caption:  "Test reel",
			Privacy:  models.PrivacySettingPublic,
			Author: models.PostAuthor{
				ID:       userID.Hex(),
				Username: "testuser",
			},
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		result, err := repo.CreateReel(context.Background(), reel)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reel.VideoURL, result.VideoURL)
		assert.False(t, result.CreatedAt.IsZero())
		assert.False(t, result.UpdatedAt.IsZero())
	})

	mt.Run("error", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reel := &models.Reel{
			VideoURL: "https://example.com/video.mp4",
		}

		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   0,
			Code:    11000,
			Message: "duplicate key error",
		}))

		result, err := repo.CreateReel(context.Background(), reel)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetReelByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		expectedReel := models.Reel{
			ID:       reelID,
			VideoURL: "https://example.com/video.mp4",
			Caption:  "Test reel",
			Privacy:  models.PrivacySettingPublic,
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.reels", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: reelID},
			{Key: "video_url", Value: expectedReel.VideoURL},
			{Key: "caption", Value: expectedReel.Caption},
			{Key: "privacy", Value: expectedReel.Privacy},
		}))

		result, err := repo.GetReelByID(context.Background(), reelID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reelID, result.ID)
		assert.Equal(t, expectedReel.VideoURL, result.VideoURL)
	})

	mt.Run("not found", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reelID := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, "db.reels", mtest.FirstBatch))

		result, err := repo.GetReelByID(context.Background(), reelID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetUserReels(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		userID := primitive.NewObjectID()
		reel1ID := primitive.NewObjectID()
		reel2ID := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "db.reels", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: reel1ID},
			{Key: "user_id", Value: userID},
			{Key: "caption", Value: "Reel 1"},
		})
		second := mtest.CreateCursorResponse(1, "db.reels", mtest.NextBatch, bson.D{
			{Key: "_id", Value: reel2ID},
			{Key: "user_id", Value: userID},
			{Key: "caption", Value: "Reel 2"},
		})
		killCursors := mtest.CreateCursorResponse(0, "db.reels", mtest.NextBatch)

		mt.AddMockResponses(first, second, killCursors)

		reels, err := repo.GetUserReels(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, reels, 2)
	})

	mt.Run("empty", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		userID := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, "db.reels", mtest.FirstBatch))

		reels, err := repo.GetUserReels(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, reels)
	})
}

func TestDeleteReel(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "acknowledged", Value: true},
			{Key: "n", Value: 1},
		})

		err := repo.DeleteReel(context.Background(), reelID, userID)

		assert.NoError(t, err)
	})
}

func TestIncrementViews(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reelID := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.IncrementViews(context.Background(), reelID)

		assert.NoError(t, err)
	})
}

func TestGetReelsFeed(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("public reels", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		userID := primitive.NewObjectID()
		reelID := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "db.reels", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: reelID},
			{Key: "privacy", Value: "PUBLIC"},
			{Key: "caption", Value: "Public reel"},
		})
		killCursors := mtest.CreateCursorResponse(0, "db.reels", mtest.NextBatch)

		mt.AddMockResponses(first, killCursors)

		reels, err := repo.GetReelsFeed(context.Background(), userID, []primitive.ObjectID{}, 10, 0)

		require.NoError(t, err)
		assert.Len(t, reels, 1)
		assert.Equal(t, models.PrivacySettingType("PUBLIC"), reels[0].Privacy)
	})
}

func TestAddComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection:         mt.Coll,
			commentsCollection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		comment := models.Comment{
			ID:        primitive.NewObjectID(),
			UserID:    primitive.NewObjectID(),
			Content:   "Great reel!",
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.AddComment(context.Background(), reelID, comment)

		assert.NoError(t, err)
	})
}

func TestGetComments(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			commentsCollection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		commentID := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "db.comments", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: commentID},
			{Key: "reel_id", Value: reelID},
			{Key: "content", Value: "Test comment"},
		})
		killCursors := mtest.CreateCursorResponse(0, "db.comments", mtest.NextBatch)

		mt.AddMockResponses(first, killCursors)

		comments, err := repo.GetComments(context.Background(), reelID, 20, 0)

		require.NoError(t, err)
		assert.Len(t, comments, 1)
	})
}

func TestAddReply(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			commentsCollection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		commentID := primitive.NewObjectID()
		reply := models.Reply{
			ID:        primitive.NewObjectID(),
			CommentID: commentID,
			UserID:    primitive.NewObjectID(),
			Content:   "Nice comment!",
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.AddReply(context.Background(), reelID, commentID, reply)

		assert.NoError(t, err)
	})
}

func TestGetReaction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		repo := &ReelRepository{
			reactionsCollection: mt.Coll,
		}

		targetID := primitive.NewObjectID()
		userID := primitive.NewObjectID()
		reactionID := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.reactions", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: reactionID},
			{Key: "target_id", Value: targetID},
			{Key: "user_id", Value: userID},
			{Key: "type", Value: "LIKE"},
		}))

		reaction, err := repo.GetReaction(context.Background(), targetID, userID)

		require.NoError(t, err)
		assert.NotNil(t, reaction)
		assert.Equal(t, models.ReactionType("LIKE"), reaction.Type)
	})

	mt.Run("not found", func(mt *mtest.T) {
		repo := &ReelRepository{
			reactionsCollection: mt.Coll,
		}

		targetID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, "db.reactions", mtest.FirstBatch))

		reaction, err := repo.GetReaction(context.Background(), targetID, userID)

		assert.NoError(t, err)
		assert.Nil(t, reaction)
	})
}

func TestAddReaction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection:          mt.Coll,
			reactionsCollection: mt.Coll,
		}

		reaction := &models.Reaction{
			ID:         primitive.NewObjectID(),
			UserID:     primitive.NewObjectID(),
			TargetID:   primitive.NewObjectID(),
			TargetType: "reel",
			Type:       models.ReactionLike,
			CreatedAt:  time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.AddReaction(context.Background(), reaction)

		assert.NoError(t, err)
	})
}

func TestRemoveReaction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection:          mt.Coll,
			reactionsCollection: mt.Coll,
		}

		reaction := &models.Reaction{
			ID:         primitive.NewObjectID(),
			UserID:     primitive.NewObjectID(),
			TargetID:   primitive.NewObjectID(),
			TargetType: "reel",
			Type:       models.ReactionLike,
		}

		// Delete reaction
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "acknowledged", Value: true},
			{Key: "n", Value: 1},
		})
		// Update reel counts
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.RemoveReaction(context.Background(), reaction)

		assert.NoError(t, err)
	})
}

func TestReactToComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			commentsCollection: mt.Coll,
		}

		reelID := primitive.NewObjectID()
		commentID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 0},
		})
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 1},
		})

		err := repo.ReactToComment(context.Background(), reelID, commentID, userID, models.ReactionLike)

		assert.NoError(t, err)
	})
}

func TestListReels(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		reelID := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "db.reels", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: reelID},
			{Key: "caption", Value: "Test reel"},
		})
		killCursors := mtest.CreateCursorResponse(0, "db.reels", mtest.NextBatch)

		mt.AddMockResponses(first, killCursors)

		reels, err := repo.ListReels(context.Background(), 10, 0)

		require.NoError(t, err)
		assert.Len(t, reels, 1)
	})
}

func TestUpdateAuthorInfo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := &ReelRepository{
			collection: mt.Coll,
		}

		userID := primitive.NewObjectID()
		author := models.PostAuthor{
			ID:       userID.Hex(),
			Username: "newusername",
			Avatar:   "newavatar.png",
			FullName: "New Name",
		}

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 5},
		})

		err := repo.UpdateAuthorInfo(context.Background(), userID, author)

		assert.NoError(t, err)
	})
}
