package ports

import (
	"context"
	"job-schedular-service/internal/core/domain"
	"time"
)

// QueuePort defines the contract for job queue operations
type QueuePort interface {
	// Enqueue adds a job to the appropriate queue based on priority and schedule
	Enqueue(ctx context.Context, job *domain.Job) error

	// Dequeue gets the next available job from queues (respects priority)
	Dequeue(ctx context.Context, priorities []domain.Priority) (*domain.Job, error)

	// EnqueueDelayed adds job to delayed/scheduled queue
	EnqueueDelayed(ctx context.Context, job *domain.Job, delayUntil time.Time) error

	// MoveToDeadLetter sends failed job to dead letter queue
	MoveToDeadLetter(ctx context.Context, job *domain.Job) error

	// GetQueueStats returns current queue statistics
	GetQueueStats(ctx context.Context) (QueueStats, error)

	// ProcessDelayedJobs moves scheduled jobs to active queues when ready
	ProcessDelayedJobs(ctx context.Context) error
}

type QueueStats struct {
	ActiveJobs     map[domain.Priority]int64 `json:"active_jobs"`
	DelayedJobs    int64                     `json:"delayed_jobs"`
	DeadLetterJobs int64                     `json:"dead_letter_jobs"`
}
