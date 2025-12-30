package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	DelayedKey = "notifications:delayed"
)

// Service manages delayed event scheduling using Redis ZSET
type Service struct {
	redis  *redis.Client
	queue  adapters.QueueAdapter
	logger *zap.Logger
}

func NewService(redis *redis.Client, queue adapters.QueueAdapter, logger *zap.Logger) *Service {
	return &Service{
		redis:  redis,
		queue:  queue,
		logger: logger,
	}
}

// Schedule adds an event to the delayed set with a future timestamp score
func (s *Service) Schedule(ctx context.Context, event *adapters.NotificationEvent, delay time.Duration) error {
	if s.redis == nil {
		// Fallback for no-redis mode: direct sleep (blocking, bad but safe for tests)
		// Or separate goroutine
		go func() {
			time.Sleep(delay)
			s.queue.Publish(context.Background(), event)
		}()
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	score := float64(time.Now().Add(delay).UnixNano()) / 1e9
	member := redis.Z{
		Score:  score,
		Member: data,
	}

	return s.redis.ZAdd(ctx, DelayedKey, member).Err()
}

// StartPoller continuously checks for due events and moves them to the main queue
func (s *Service) StartPoller(ctx context.Context) {
	if s.redis == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	s.logger.Info("🕰️ Delayed Retry Scheduler Started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processDueEvents(ctx)
		}
	}
}

func (s *Service) processDueEvents(ctx context.Context) {
	// Atomic move from Delayed ZSET to Active List (durability)
	// If the worker crashes after this script but before publishing, items remain in 'active' list
	// A separate recovery process (or startup check) could re-queue them.
	// For now, this is "At Least Once" if we just process the list.
	script := redis.NewScript(`
		local key = KEYS[1]
		local activeList = KEYS[2]
		local maxScore = ARGV[1]
		local limit = tonumber(ARGV[2])

		local events = redis.call('ZRANGEBYSCORE', key, '-inf', maxScore, 'LIMIT', 0, limit)
		if #events > 0 then
			redis.call('ZREM', key, unpack(events))
			// Push to active list to persist them while processing
			// In a full implementation, we'd use RPOPLPUSH per item, but batch move is efficient
			for i, v in ipairs(events) do
				redis.call('RPUSH', activeList, v)
			end
		end
		return events
	`)

	now := float64(time.Now().UnixNano()) / 1e9
	activeKey := "notifications:active" // Durable list
	res, err := script.Run(ctx, s.redis, []string{DelayedKey, activeKey}, now, 20).Result()
	if err != nil && err != redis.Nil {
		s.logger.Error("Failed to poll delayed events", zap.Error(err))
		return
	}

	eventsString, ok := res.([]interface{})
	if !ok || len(eventsString) == 0 {
		return
	}

	for _, dataStr := range eventsString {
		data := []byte(dataStr.(string))
		var event adapters.NotificationEvent
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("Failed to unmarshal delayed event", zap.Error(err))
			// Poison message? Remove from list to avoid block?
			// For now, we leave it or remove it. Ideally remove.
			s.redis.LRem(ctx, activeKey, 1, dataStr)
			continue
		}

		s.logger.Info("⏰ Triggering delayed retry", zap.String("id", event.ID))
		if err := s.queue.Publish(ctx, &event); err != nil {
			s.logger.Error("Failed to publish delayed event", zap.Error(err))
			// Re-queue with backoff?
			// Use ZADD to put back in delayed (retry later)
			// And remove from active list
			s.redis.ZAdd(ctx, DelayedKey, redis.Z{
				Score:  float64(time.Now().Add(10*time.Second).UnixNano()) / 1e9,
				Member: data,
			})
			s.redis.LRem(ctx, activeKey, 1, dataStr)
		} else {
			// Success: Remove from active list
			s.redis.LRem(ctx, activeKey, 1, dataStr)
		}
	}
}
