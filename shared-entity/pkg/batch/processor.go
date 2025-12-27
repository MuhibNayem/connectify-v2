package batch

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Processor handles buffering and batch processing of items
type Processor[T any] struct {
	batchSize     int
	flushInterval time.Duration
	processFunc   func(context.Context, []T) error

	buffer []T
	mu     sync.Mutex

	shutdownChan chan struct{}
	wg           sync.WaitGroup
	ticker       *time.Ticker

	logger *slog.Logger
}

// Config holds configuration for the processor
type Config struct {
	BatchSize     int
	FlushInterval time.Duration
	Logger        *slog.Logger
}

// NewProcessor creates a new batch processor
func NewProcessor[T any](cfg Config, processFunc func(context.Context, []T) error) *Processor[T] {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	p := &Processor[T]{
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		processFunc:   processFunc,
		buffer:        make([]T, 0, cfg.BatchSize),
		shutdownChan:  make(chan struct{}),
		logger:        cfg.Logger,
		ticker:        time.NewTicker(cfg.FlushInterval),
	}

	p.start()
	return p
}

func (p *Processor[T]) start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.ticker.C:
				p.flushSafe()
			case <-p.shutdownChan:
				p.flushSafe()
				return
			}
		}
	}()
}

// Add adds an item to the processor
func (p *Processor[T]) Add(ctx context.Context, item T) error {
	p.mu.Lock()
	p.buffer = append(p.buffer, item)
	shouldFlush := len(p.buffer) >= p.batchSize
	p.mu.Unlock()

	if shouldFlush {
		// Reset ticker to avoid double flushing
		p.ticker.Reset(p.flushInterval)
		return p.flushSafe()
	}

	return nil
}

// flushSafe wraps flush with mutex protection for public calling
func (p *Processor[T]) flushSafe() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flush(context.Background())
}

// flush sends the current buffer to processing
// Caller must hold lock
func (p *Processor[T]) flush(ctx context.Context) error {
	if len(p.buffer) == 0 {
		return nil
	}

	items := p.buffer
	// Allocate new buffer to release lock quickly
	p.buffer = make([]T, 0, p.batchSize)

	// Process logic should not hold lock
	// We release lock here? No, we need to carefully swap.
	// We ALREADY swapped p.buffer above.
	// So we can technically release lock, process, but for simplicity
	// let's do async processing? NO, blocking is safer for backpressure.
	// But holding lock blocks Add().
	// Correct pattern: Swap buffer, Unlock, Process.

	// BUT, wait. My flush is called from Add() (holding lock? No, Add releases lock).
	// Add() releases lock BEFORE calling flushSafe().
	// flushSafe acquires lock.
	// So inside flush(), we hold lock.

	// Optimization: Swap and Unlock, then Process.
	// But we need to ensure order? Batch processing order might matter.
	// With concurrent Add(), we might have race if we unlock.
	// Actually, strictly sequential flush is better.
	// If we unlock, another flush might happen before this one finishes.
	// Let's keep it simple: Process synchronously.
	// Ideally processFunc is fast (e.g. queueing to DB).

	// ACTUALLY: Best practice for batch processors is to NOT block Add().
	// But for this simplified version, let's keep it robust.
	// We will UNLOCK before processing to prevent blocking Add().
	// We need a separate 'processing' lock if we want to ensure sequential processing.

	// For this specific use case (View Counts), order strictly doesn't matter (summing).
	// But let's stay safe: Copy buffer, Clear buffer, Release Lock. Process.

	processingItems := items
	// Unlock is tricky because 'defer' in flushSafe.
	// We cannot unlock here if flushSafe deferred unlock.
	// Refactor logic: flushSafe should Swap, Unlock, Process.

	// Since I can't easily change the structure without rewriting helper methods:
	// Let's modify flushSafe logic here:
	// It's called inside 'flushSafe' which holds lock.
	// I'll just run process in a goroutine? No, we want error reporting.
	// But Add() returns error.
	// If the DB is slow, Add() blocks. This provides backpressure. Good.

	err := p.processFunc(ctx, processingItems)
	if err != nil {
		p.logger.Error("Batch processing failed", "error", err, "count", len(processingItems))
		// Optional: Retry logic or DLQ?
		// For now, log.
	}
	return err
}

// Stop gracefully shuts down the processor, flushing remaining items
func (p *Processor[T]) Stop() {
	p.ticker.Stop()
	close(p.shutdownChan)
	p.wg.Wait()
}
