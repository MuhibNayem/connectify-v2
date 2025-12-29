package cache

import (
	"context"
	"encoding/json"
	"log/slog"

	pkgredis "github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	invalidationChannel = "friendship:cache:invalidate"
)

// InvalidationMessage represents a cache invalidation event
type InvalidationMessage struct {
	Type   string `json:"type"`
	UserA  string `json:"user_a"`
	UserB  string `json:"user_b"`
	Action string `json:"action"` // "invalidate", "update"
}

// PubSub handles distributed cache invalidation via Redis Pub/Sub
type PubSub struct {
	client *pkgredis.ClusterClient
	cache  *FriendshipCache
	logger *slog.Logger
	pubsub *redis.PubSub
}

// NewPubSub creates a new PubSub handler
func NewPubSub(client *pkgredis.ClusterClient, cache *FriendshipCache, logger *slog.Logger) *PubSub {
	if logger == nil {
		logger = slog.Default()
	}
	return &PubSub{
		client: client,
		cache:  cache,
		logger: logger,
	}
}

// PublishInvalidation broadcasts cache invalidation to all instances
func (p *PubSub) PublishInvalidation(ctx context.Context, userA, userB primitive.ObjectID) error {
	msg := InvalidationMessage{
		Type:   "friendship",
		UserA:  userA.Hex(),
		UserB:  userB.Hex(),
		Action: "invalidate",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, invalidationChannel, string(data))
}

// Subscribe starts listening for cache invalidation events
func (p *PubSub) Subscribe(ctx context.Context) {
	p.pubsub = p.client.Subscribe(ctx, invalidationChannel)

	// Wait for subscription confirmation
	_, err := p.pubsub.Receive(ctx)
	if err != nil {
		p.logger.Error("Failed to subscribe to cache invalidation channel", "error", err)
		return
	}

	p.logger.Info("Subscribed to cache invalidation channel", "channel", invalidationChannel)

	go p.listen(ctx)
}

// listen processes incoming invalidation messages
func (p *PubSub) listen(ctx context.Context) {
	ch := p.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Cache invalidation listener stopping")
			return

		case msg, ok := <-ch:
			if !ok {
				p.logger.Warn("Cache invalidation channel closed")
				return
			}

			p.handleMessage(ctx, msg)
		}
	}
}

func (p *PubSub) handleMessage(ctx context.Context, msg *redis.Message) {
	var invalidation InvalidationMessage
	if err := json.Unmarshal([]byte(msg.Payload), &invalidation); err != nil {
		p.logger.Error("Failed to parse invalidation message", "error", err, "payload", msg.Payload)
		return
	}

	userA, err := primitive.ObjectIDFromHex(invalidation.UserA)
	if err != nil {
		return
	}
	userB, err := primitive.ObjectIDFromHex(invalidation.UserB)
	if err != nil {
		return
	}

	p.logger.Debug("Received cache invalidation",
		"user_a", invalidation.UserA,
		"user_b", invalidation.UserB,
		"action", invalidation.Action,
	)

	// Invalidate local cache
	if p.cache != nil {
		p.cache.InvalidateFriendship(ctx, userA, userB)
	}
}

// Close shuts down the pub/sub connection
func (p *PubSub) Close() error {
	if p.pubsub != nil {
		return p.pubsub.Close()
	}
	return nil
}
