// internal/core/services/processor.go
package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"time"
)

// ProcessorService handles job execution and lifecycle management
type ProcessorService struct {
	storage  ports.StoragePort
	queue    ports.QueuePort
	metrics  ports.MetricsPort
	logger   ports.LoggerPort
	handlers map[domain.JobType]JobHandler
}

// JobHandler defines the interface for job execution
type JobHandler interface {
	Handle(ctx context.Context, job *domain.Job) error
	GetType() domain.JobType
}

func NewProcessorService(
	storage ports.StoragePort,
	queue ports.QueuePort,
	metrics ports.MetricsPort,
	logger ports.LoggerPort,
) *ProcessorService {
	return &ProcessorService{
		storage:  storage,
		queue:    queue,
		metrics:  metrics,
		logger:   logger,
		handlers: make(map[domain.JobType]JobHandler),
	}
}

func (p *ProcessorService) RegisterHandler(handler JobHandler) {
	p.handlers[handler.GetType()] = handler
}

// ProcessNextJob gets and processes the next available job
func (p *ProcessorService) ProcessNextJob(ctx context.Context, workerID string, priorities []domain.Priority) error {
	// Get next job from queue
	job, err := p.queue.Dequeue(ctx, priorities)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	if job == nil {
		return fmt.Errorf("no jobs available")
	}

	// Process the job
	return p.ProcessJob(ctx, job, workerID)
}

// ProcessJob executes a specific job
func (p *ProcessorService) ProcessJob(ctx context.Context, job *domain.Job, workerID string) error {
	// Mark job as processing
	job.MarkAsProcessing(workerID)
	if err := p.storage.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Record job history
	history := &ports.JobHistory{
		ID:            uuid.New().String(),
		JobID:         job.ID,
		WorkerID:      workerID,
		AttemptNumber: job.RetryPolicy.CurrentAttempts + 1,
		StartedAt:     time.Now(),
		Status:        "processing",
	}

	startTime := time.Now()

	// Execute job with timeout
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute) // TODO: make configurable
	defer cancel()

	err := p.executeJob(jobCtx, job)
	duration := time.Since(startTime)

	// Update history
	history.Duration = duration
	now := time.Now()
	history.CompletedAt = &now

	if err != nil {
		p.handleJobFailure(ctx, job, history, err)
	} else {
		p.handleJobSuccess(ctx, job, history)
	}

	// Save history
	if saveErr := p.storage.SaveJobHistory(ctx, history); saveErr != nil {
		p.logger.Error(ctx, "Failed to save job history",
			ports.String("job_id", job.ID.String()),
			ports.String("error", saveErr.Error()),
		)
	}

	return err
}

func (p *ProcessorService) executeJob(ctx context.Context, job *domain.Job) error {
	handler, exists := p.handlers[job.Type]
	if !exists {
		return fmt.Errorf("no handler registered for job type: %s", job.Type)
	}

	return handler.Handle(ctx, job)
}

func (p *ProcessorService) handleJobSuccess(ctx context.Context, job *domain.Job, history *ports.JobHistory) {
	job.MarkAsCompleted()
	history.Status = "completed"

	if err := p.storage.UpdateJob(ctx, job); err != nil {
		p.logger.Error(ctx, "Failed to update completed job",
			ports.String("job_id", job.ID.String()),
			ports.String("error", err.Error()),
		)
	}

	p.metrics.IncrementJobsCompleted(job.Type, job.Priority, history.Duration)

	p.logger.Info(ctx, "Job completed successfully",
		ports.String("job_id", job.ID.String()),
		ports.Duration("duration", history.Duration),
	)
}

func (p *ProcessorService) handleJobFailure(ctx context.Context, job *domain.Job, history *ports.JobHistory, err error) {
	job.MarkAsFailed(err)
	history.Status = "failed"
	history.Error = err.Error()

	if job.CanRetry() {
		// Schedule retry
		job.Status = domain.StatusScheduled
		job.ScheduledAt = job.NextRetryAt()

		// Re-enqueue for retry
		if enqueueErr := p.queue.EnqueueDelayed(ctx, job, job.ScheduledAt); enqueueErr != nil {
			p.logger.Error(ctx, "Failed to enqueue job for retry",
				ports.String("job_id", job.ID.String()),
				ports.String("error", enqueueErr.Error()),
			)
		}

		p.metrics.IncrementJobsRetried(job.Type, job.Priority)

		p.logger.Warn(ctx, "Job failed, scheduled for retry",
			ports.String("job_id", job.ID.String()),
			ports.Int("attempt", job.RetryPolicy.CurrentAttempts),
			ports.Int("max_attempts", job.RetryPolicy.MaxAttempts),
			ports.String("error", err.Error()),
		)
	} else {
		// Send to dead letter queue
		job.MarkAsDead()

		if dlqErr := p.queue.MoveToDeadLetter(ctx, job); dlqErr != nil {
			p.logger.Error(ctx, "Failed to move job to dead letter queue",
				ports.String("job_id", job.ID.String()),
				ports.String("error", dlqErr.Error()),
			)
		}

		p.metrics.IncrementJobsDeadLettered(job.Type, job.Priority)

		p.logger.Error(ctx, "Job moved to dead letter queue",
			ports.String("job_id", job.ID.String()),
			ports.String("error", err.Error()),
		)
	}

	p.metrics.IncrementJobsFailed(job.Type, job.Priority, err.Error())

	if err := p.storage.UpdateJob(ctx, job); err != nil {
		p.logger.Error(ctx, "Failed to update failed job",
			ports.String("job_id", job.ID.String()),
			ports.String("error", err.Error()),
		)
	}
}
