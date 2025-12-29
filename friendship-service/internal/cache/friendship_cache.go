package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/singleflight"
)

const (
	friendshipKeyPrefix = "friendship:status:"
	blockKeyPrefix      = "friendship:block:"
	defaultTTL          = 5 * time.Minute
)

// FriendshipCache provides optimized caching for friendship data
// with singleflight to prevent cache stampede
type FriendshipCache struct {
	client *redis.ClusterClient
	ttl    time.Duration
	sf     singleflight.Group // Prevents duplicate DB calls on cache miss
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

// Key generation with object pooling
var keyBuilderPool = sync.Pool{
	New: func() interface{} {
		return &keyBuilder{buf: make([]byte, 0, 128)}
	},
}

type keyBuilder struct {
	buf []byte
}

func (kb *keyBuilder) reset() {
	kb.buf = kb.buf[:0]
}

func friendshipKey(userID1, userID2 primitive.ObjectID) string {
	// Ensure consistent key ordering for bidirectional lookup
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

// GetOrLoadFriendshipStatus uses singleflight to prevent cache stampede
// Only one goroutine will load from DB for the same key, others wait for result
func (c *FriendshipCache) GetOrLoadFriendshipStatus(
	ctx context.Context,
	userID1, userID2 primitive.ObjectID,
	loader func(context.Context) (*models.FriendshipStatus, error),
) (*models.FriendshipStatus, error) {
	key := friendshipKey(userID1, userID2)

	// Try cache first
	cached, err := c.GetFriendshipStatus(ctx, userID1, userID2)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Singleflight: only one goroutine loads from DB
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// Double-check cache (another goroutine might have populated it)
		if cached, err := c.GetFriendshipStatus(ctx, userID1, userID2); err == nil && cached != nil {
			return cached, nil
		}

		// Load from database
		status, err := loader(ctx)
		if err != nil {
			return nil, err
		}

		// Populate cache asynchronously
		if status != nil {
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = c.SetFriendshipStatus(cacheCtx, userID1, userID2, *status)
			}()
		}

		return status, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.FriendshipStatus), nil
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

// GetOrLoadBlockStatus uses singleflight to prevent cache stampede for block status
func (c *FriendshipCache) GetOrLoadBlockStatus(
	ctx context.Context,
	blockerID, blockedID primitive.ObjectID,
	loader func(context.Context) (*bool, error),
) (*bool, error) {
	key := blockKey(blockerID, blockedID)

	// Try cache first
	cached, err := c.GetBlockStatus(ctx, blockerID, blockedID)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Singleflight
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		if cached, err := c.GetBlockStatus(ctx, blockerID, blockedID); err == nil && cached != nil {
			return cached, nil
		}

		status, err := loader(ctx)
		if err != nil {
			return nil, err
		}

		if status != nil {
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = c.SetBlockStatus(cacheCtx, blockerID, blockedID, *status)
			}()
		}

		return status, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*bool), nil
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

// BatchGetFriendshipStatuses retrieves multiple friendship statuses in one call
func (c *FriendshipCache) BatchGetFriendshipStatuses(
	ctx context.Context,
	userID primitive.ObjectID,
	otherIDs []primitive.ObjectID,
) (map[primitive.ObjectID]*models.FriendshipStatus, []primitive.ObjectID) {
	results := make(map[primitive.ObjectID]*models.FriendshipStatus, len(otherIDs))
	misses := make([]primitive.ObjectID, 0, len(otherIDs))

	for _, otherID := range otherIDs {
		status, err := c.GetFriendshipStatus(ctx, userID, otherID)
		if err == nil && status != nil {
			results[otherID] = status
		} else {
			misses = append(misses, otherID)
		}
	}

	return results, misses
}
