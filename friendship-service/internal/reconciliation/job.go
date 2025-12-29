package reconciliation

import (
	"context"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/cache"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Job performs periodic reconciliation between MongoDB and Neo4j
type Job struct {
	friendshipRepo *repository.FriendshipRepository
	graphRepo      *repository.GraphRepository
	cache          *cache.FriendshipCache
	metrics        *metrics.BusinessMetrics
	logger         *slog.Logger
	interval       time.Duration
	batchSize      int64
	stopChan       chan struct{}
}

// NewJob creates a new reconciliation job
func NewJob(
	friendshipRepo *repository.FriendshipRepository,
	graphRepo *repository.GraphRepository,
	cache *cache.FriendshipCache,
	metrics *metrics.BusinessMetrics,
	logger *slog.Logger,
) *Job {
	if logger == nil {
		logger = slog.Default()
	}
	return &Job{
		friendshipRepo: friendshipRepo,
		graphRepo:      graphRepo,
		cache:          cache,
		metrics:        metrics,
		logger:         logger,
		interval:       1 * time.Hour,
		batchSize:      1000,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the reconciliation job
func (j *Job) Start(ctx context.Context) {
	j.logger.Info("Starting reconciliation job", "interval", j.interval)

	// Run immediately on start
	j.reconcile(ctx)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info("Reconciliation job stopping (context cancelled)")
			return
		case <-j.stopChan:
			j.logger.Info("Reconciliation job stopping (stop signal)")
			return
		case <-ticker.C:
			j.reconcile(ctx)
		}
	}
}

// Stop signals the job to stop
func (j *Job) Stop() {
	close(j.stopChan)
}

func (j *Job) reconcile(ctx context.Context) {
	if j.graphRepo == nil {
		j.logger.Debug("Skipping reconciliation - no graph repository")
		return
	}

	j.logger.Info("Starting reconciliation run")
	start := time.Now()

	repaired := 0
	checked := 0

	// Get all accepted friendships from MongoDB
	friendships, err := j.friendshipRepo.GetAllAccepted(ctx, j.batchSize)
	if err != nil {
		j.logger.Error("Failed to fetch friendships for reconciliation", "error", err)
		return
	}

	for _, friendship := range friendships {
		checked++

		// Check if Neo4j matches
		neoAreFriends, err := j.graphRepo.AreFriends(ctx, friendship.RequesterID, friendship.ReceiverID)
		if err != nil {
			j.logger.Warn("Failed to check Neo4j friendship",
				"requester", friendship.RequesterID.Hex(),
				"receiver", friendship.ReceiverID.Hex(),
				"error", err,
			)
			continue
		}

		if friendship.Status == models.FriendshipStatusAccepted && !neoAreFriends {
			// INCONSISTENCY: MongoDB says friends, Neo4j says not
			j.logger.Warn("Inconsistency detected - repairing",
				"friendship_id", friendship.ID.Hex(),
				"mongo_status", friendship.Status,
				"neo4j_are_friends", neoAreFriends,
			)

			// Repair Neo4j
			if err := j.graphRepo.AcceptRequest(ctx, friendship.RequesterID, friendship.ReceiverID); err != nil {
				j.logger.Error("Failed to repair Neo4j", "error", err)
				continue
			}

			// Invalidate cache
			if j.cache != nil {
				j.cache.InvalidateFriendship(ctx, friendship.RequesterID, friendship.ReceiverID)
			}

			// Record metric
			if j.metrics != nil {
				j.metrics.RecordInconsistency("mongo_neo4j_mismatch")
			}

			repaired++
		}
	}

	j.logger.Info("Reconciliation run completed",
		"checked", checked,
		"repaired", repaired,
		"duration", time.Since(start),
	)
}

// RepairOnRead performs read-repair when inconsistency is detected during a read
func (j *Job) RepairOnRead(ctx context.Context, userA, userB primitive.ObjectID, mongoResult bool) {
	if j.graphRepo == nil {
		return
	}

	go func() {
		var err error
		if mongoResult {
			// MongoDB says friends, ensure Neo4j agrees
			err = j.graphRepo.AcceptRequest(ctx, userA, userB)
		} else {
			// MongoDB says not friends, ensure Neo4j agrees
			err = j.graphRepo.Unfriend(ctx, userA, userB)
		}

		if err != nil {
			j.logger.Error("Read repair failed",
				"user_a", userA.Hex(),
				"user_b", userB.Hex(),
				"mongo_result", mongoResult,
				"error", err,
			)
		} else {
			j.logger.Info("Read repair successful",
				"user_a", userA.Hex(),
				"user_b", userB.Hex(),
			)

			if j.metrics != nil {
				j.metrics.RecordInconsistency("read_repair_triggered")
			}
		}
	}()
}
