package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/spydotech-group/shared-entity/models"
)

type CacheRepository struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewCacheRepository(addrs []string, password string) *CacheRepository {
	var client redis.UniversalClient

	if len(addrs) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addrs,
			Password: password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     addrs[0],
			Password: password,
		})
	}

	return &CacheRepository{
		client: client,
		ttl:    1 * time.Hour, // Default cache TTL
	}
}

// ----------------------------- Post Caching -----------------------------

func (r *CacheRepository) SetPost(ctx context.Context, post *models.Post) error {
	data, err := json.Marshal(post)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("post:%s", post.ID.Hex())
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *CacheRepository) GetPost(ctx context.Context, postID string) (*models.Post, error) {
	key := fmt.Sprintf("post:%s", postID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var post models.Post
	if err := json.Unmarshal(data, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *CacheRepository) GetPosts(ctx context.Context, postIDs []string) ([]*models.Post, []string, error) {
	if len(postIDs) == 0 {
		return []*models.Post{}, []string{}, nil
	}

	keys := make([]string, len(postIDs))
	for i, id := range postIDs {
		keys[i] = fmt.Sprintf("post:%s", id)
	}

	// MGET
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, nil, err
	}

	posts := make([]*models.Post, 0, len(postIDs))
	missingIDs := []string{}

	for i, result := range results {
		if result == nil {
			missingIDs = append(missingIDs, postIDs[i])
			continue
		}

		strData, ok := result.(string)
		if !ok {
			missingIDs = append(missingIDs, postIDs[i])
			continue
		}

		var post models.Post
		if err := json.Unmarshal([]byte(strData), &post); err != nil {
			missingIDs = append(missingIDs, postIDs[i])
			continue
		}
		posts = append(posts, &post)
	}

	return posts, missingIDs, nil
}

func (r *CacheRepository) InvalidatePost(ctx context.Context, postID string) error {
	key := fmt.Sprintf("post:%s", postID)
	return r.client.Del(ctx, key).Err()
}

// ----------------------------- Timeline (Fan-out) -----------------------------

func (r *CacheRepository) PushToTimeline(ctx context.Context, userID, postID string) error {
	key := fmt.Sprintf("timeline:%s", userID)
	pipe := r.client.Pipeline()
	// Push to head
	pipe.LPush(ctx, key, postID)
	// Trim to keep only last 500 posts (Cost efficiency)
	pipe.LTrim(ctx, key, 0, 499)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *CacheRepository) GetTimeline(ctx context.Context, userID string, offset, limit int64) ([]string, error) {
	key := fmt.Sprintf("timeline:%s", userID)
	return r.client.LRange(ctx, key, offset, offset+limit-1).Result()
}
