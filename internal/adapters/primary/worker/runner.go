package worker

import (
	"context"
	"fmt"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"job-schedular-service/internal/core/services"
	"sync"
	"time"
)

// WorkerRunner manages job processing workers
type WorkerRunner struct {
	processor     *services.ProcessorService
	storage       ports.StoragePort
	logger        ports.LoggerPort
	workerID      string
	hostname      string
	maxConcurrent int
	priorities    []domain.Priority
	currentJobs   int64
	mu            sync.RWMutex
}

func NewWorkerRunner(
	processor *services.ProcessorService,
	storage ports.StoragePort,
	logger ports.LoggerPort,
	workerID string,
	hostname string,
	maxConcurrent int,
) *WorkerRunner {
	return &WorkerRunner{
		processor:     processor,
		storage:       storage,
		logger:        logger,
		workerID:      workerID,
		hostname:      hostname,
		maxConcurrent: maxConcurrent,
		priorities:    []domain.Priority{domain.PriorityCritical, domain.PriorityHigh, domain.PriorityMedium, domain.PriorityLow},
	}
}

func (w *WorkerRunner) Start(ctx context.Context) error {
	// Register worker
	worker := &ports.Worker{
		ID:             w.workerID,
		Hostname:       w.hostname,
		SupportedTypes: []domain.JobType{domain.JobTypeEmail, domain.JobTypePayment, domain.JobTypeDataExport},
		MaxConcurrent:  w.maxConcurrent,
		CurrentJobs:    0,
		Status:         "active",
		LastHeartbeat:  time.Now(),
		RegisteredAt:   time.Now(),
	}

	if err := w.storage.RegisterWorker(ctx, worker); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	w.logger.Info(ctx, "Worker started",
		ports.String("worker_id", w.workerID),
		ports.String("hostname", w.hostname),
		ports.Int("max_concurrent", w.maxConcurrent),
	)

	var wg sync.WaitGroup

	// Start heartbeat goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.sendHeartbeat(ctx)
	}()

	// Start job processing goroutines
	for i := 0; i < w.maxConcurrent; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			w.processJobs(ctx, fmt.Sprintf("%s-%d", w.workerID, workerNum))
		}(i)
	}

	// Wait for shutdown
	<-ctx.Done()
	w.logger.Info(ctx, "Worker shutting down", ports.String("worker_id", w.workerID))

	wg.Wait()
	return nil
}

func (w *WorkerRunner) sendHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.storage.UpdateWorkerHeartbeat(ctx, w.workerID); err != nil {
				w.logger.Error(ctx, "Failed to send heartbeat",
					ports.String("worker_id", w.workerID),
					ports.String("error", err.Error()),
				)
			}
		}
	}
}

func (w *WorkerRunner) processJobs(ctx context.Context, workerInstanceID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Process next available job
			err := w.processor.ProcessNextJob(ctx, workerInstanceID, w.priorities)
			if err != nil {
				// No jobs available, wait before trying again
				time.Sleep(1 * time.Second)
				continue
			}
		}
	}
}
