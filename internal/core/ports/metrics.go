package ports

import (
	"job-schedular-service/internal/core/domain"
	"time"
)

// MetricsPort defines the contract for metrics collection
type MetricsPort interface {
	// Job metrics
	IncrementJobsScheduled(jobType domain.JobType, priority domain.Priority)
	IncrementJobsCompleted(jobType domain.JobType, priority domain.Priority, duration time.Duration)
	IncrementJobsFailed(jobType domain.JobType, priority domain.Priority, reason string)
	IncrementJobsRetried(jobType domain.JobType, priority domain.Priority)
	IncrementJobsDeadLettered(jobType domain.JobType, priority domain.Priority)

	// Queue metrics
	SetQueueLength(priority domain.Priority, length int64)
	SetDelayedQueueLength(length int64)
	SetDeadLetterQueueLength(length int64)

	// Worker metrics
	SetActiveWorkers(count int64)
	IncrementWorkerHeartbeats(workerID string)
}
