package cleanup

import (
	"context"
	"log"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
)

type CleanupService struct {
	storage adapters.StorageAdapter
	ticker  *time.Ticker
	done    chan bool
}

func NewCleanupService(storage adapters.StorageAdapter, interval time.Duration) *CleanupService {
	return &CleanupService{
		storage: storage,
		ticker:  time.NewTicker(interval),
		done:    make(chan bool),
	}
}

func (c *CleanupService) Start() {
	go func() {
		for {
			select {
			case <-c.done:
				return
			case <-c.ticker.C:
				c.runCleanup()
			}
		}
	}()
}

func (c *CleanupService) Stop() {
	c.ticker.Stop()
	c.done <- true
}

func (c *CleanupService) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	count, err := c.storage.DeleteExpired(ctx)
	if err != nil {
		log.Printf("Error running cleanup: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Cleaned up %d expired notifications", count)
	}
}
