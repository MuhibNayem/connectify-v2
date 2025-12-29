package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	friendshipKeyPrefix = "friendship:status:"
	blockKeyPrefix      = "friendship:block:"
	defaultTTL          = 5 * time.Minute
)

// FriendshipCache provides caching for friendship data
type FriendshipCache struct {
	client *redis.ClusterClient
	ttl    time.Duration
}

// NewFriendshipCache creates a new FriendshipCache
func NewFriendshipCache(client *redis.ClusterClient, ttl time.Duration) *FriendshipCache {
	if ttl == 0 {
		ttl = defaultTTL
	}
	return &FriendshipCache{
		client: client,
		ttl:    ttl,
	}
}

func friendshipKey(userID1, userID2 primitive.ObjectID) string {
	// Ensure consistent key ordering
	if userID1.Hex() < userID2.Hex() {
		return fmt.Sprintf("%s%s:%s", friendshipKeyPrefix, userID1.Hex(), userID2.Hex())
	}
	return fmt.Sprintf("%s%s:%s", friendshipKeyPrefix, userID2.Hex(), userID1.Hex())
}

func blockKey(blockerID, blockedID primitive.ObjectID) string {
	return fmt.Sprintf("%s%s:%s", blockKeyPrefix, blockerID.Hex(), blockedID.Hex())
}

// GetFriendshipStatus gets the cached friendship status between two users
func (c *FriendshipCache) GetFriendshipStatus(ctx context.Context, userID1, userID2 primitive.ObjectID) (*models.FriendshipStatus, error) {
	key := friendshipKey(userID1, userID2)
	val, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil // Cache miss
	}

	var status models.FriendshipStatus
	if err := json.Unmarshal([]byte(val), &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// SetFriendshipStatus caches the friendship status between two users
func (c *FriendshipCache) SetFriendshipStatus(ctx context.Context, userID1, userID2 primitive.ObjectID, status models.FriendshipStatus) error {
	key := friendshipKey(userID1, userID2)
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, string(data), c.ttl)
}

// InvalidateFriendship invalidates the cached friendship status
func (c *FriendshipCache) InvalidateFriendship(ctx context.Context, userID1, userID2 primitive.ObjectID) error {
	key := friendshipKey(userID1, userID2)
	return c.client.Del(ctx, key)
}

// GetBlockStatus gets the cached block status
func (c *FriendshipCache) GetBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID) (*bool, error) {
	key := blockKey(blockerID, blockedID)
	val, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil // Cache miss
	}

	blocked := val == "true"
	return &blocked, nil
}

// SetBlockStatus caches the block status
func (c *FriendshipCache) SetBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID, blocked bool) error {
	key := blockKey(blockerID, blockedID)
	val := "false"
	if blocked {
		val = "true"
	}
	return c.client.Set(ctx, key, val, c.ttl)
}

// InvalidateBlockStatus invalidates the cached block status
func (c *FriendshipCache) InvalidateBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	key := blockKey(blockerID, blockedID)
	return c.client.Del(ctx, key)
}
