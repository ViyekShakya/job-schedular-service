package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"time"
)

// SchedulerService orchestrates job scheduling operations
type SchedulerService struct {
	storage ports.StoragePort
	queue   ports.QueuePort
	metrics ports.MetricsPort
	logger  ports.LoggerPort
}

func NewSchedulerService(
	storage ports.StoragePort,
	queue ports.QueuePort,
	metrics ports.MetricsPort,
	logger ports.LoggerPort,
) *SchedulerService {
	return &SchedulerService{
		storage: storage,
		queue:   queue,
		metrics: metrics,
		logger:  logger,
	}
}

type ScheduleJobRequest struct {
	Type        domain.JobType      `json:"type" validate:"required"`
	Priority    domain.Priority     `json:"priority"`
	Payload     json.RawMessage     `json:"payload" validate:"required"`
	Metadata    domain.Metadata     `json:"metadata"`
	RetryPolicy *domain.RetryPolicy `json:"retry_policy"`
	ScheduleAt  *time.Time          `json:"schedule_at"`
}

func (s *SchedulerService) ScheduleJob(ctx context.Context, req ScheduleJobRequest) (*domain.Job, error) {
	// Create job with defaults
	job := &domain.Job{
		ID:          uuid.UUID(uuid.New()),
		Type:        req.Type,
		Priority:    req.Priority,
		Payload:     req.Payload,
		Metadata:    req.Metadata,
		Status:      domain.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}

	// Set default priority if not specified
	if job.Priority == 0 {
		job.Priority = domain.PriorityMedium
	}

	// Set retry policy defaults
	if req.RetryPolicy != nil {
		job.RetryPolicy = *req.RetryPolicy
	} else {
		job.RetryPolicy = domain.NewRetryPolicy()
	}

	// Handle delayed scheduling
	if req.ScheduleAt != nil && req.ScheduleAt.After(time.Now()) {
		job.Status = domain.StatusScheduled
		job.ScheduledAt = *req.ScheduleAt
	}

	// Persist job
	if err := s.storage.SaveJob(ctx, job); err != nil {
		s.logger.Error(ctx, "Failed to save job",
			ports.String("job_id", job.ID.String()),
			ports.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Enqueue job
	if err := s.enqueueJob(ctx, job); err != nil {
		s.logger.Error(ctx, "Failed to enqueue job",
			ports.String("job_id", job.ID.String()),
			ports.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Record metrics
	s.metrics.IncrementJobsScheduled(job.Type, job.Priority)

	s.logger.Info(ctx, "Job scheduled successfully",
		ports.String("job_id", job.ID.String()),
		ports.String("type", string(job.Type)),
		ports.String("priority", job.Priority.String()),
	)

	return job, nil
}

func (s *SchedulerService) enqueueJob(ctx context.Context, job *domain.Job) error {
	if job.Status == domain.StatusScheduled {
		return s.queue.EnqueueDelayed(ctx, job, job.ScheduledAt)
	}
	return s.queue.Enqueue(ctx, job)
}

func (s *SchedulerService) GetJob(ctx context.Context, jobID uuid.UUID) (*domain.Job, error) {
	return s.storage.GetJob(ctx, jobID)
}

func (s *SchedulerService) ListJobs(ctx context.Context, filter ports.JobFilter) ([]*domain.Job, error) {
	return s.storage.ListJobs(ctx, filter)
}

func (s *SchedulerService) GetQueueStats(ctx context.Context) (ports.QueueStats, error) {
	return s.queue.GetQueueStats(ctx)
}
