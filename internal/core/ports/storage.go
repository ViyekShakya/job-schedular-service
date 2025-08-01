package ports

import (
	"context"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"time"
)

// StoragePort defines the contract for job persistence
type StoragePort interface {
	SaveJob(ctx context.Context, job *domain.Job) error
	GetJob(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) error
	ListJobs(ctx context.Context, filter JobFilter) ([]*domain.Job, error)

	SaveJobHistory(ctx context.Context, history *JobHistory) error
	GetJobHistory(ctx context.Context, jobID uuid.UUID) ([]*JobHistory, error)

	RegisterWorker(ctx context.Context, worker *Worker) error
	UpdateWorkerHeartbeat(ctx context.Context, workerID string) error
	GetActiveWorkers(ctx context.Context) ([]*Worker, error)
	DeactivateStaleWorkers(ctx context.Context, timeout time.Duration) error
}

type JobFilter struct {
	Status   *domain.Status
	JobType  *domain.JobType
	Priority *domain.Priority
	Limit    int
	Offset   int
}

type JobHistory struct {
	ID            string        `json:"id"`
	JobID         uuid.UUID     `json:"job_id"`
	WorkerID      string        `json:"worker_id"`
	AttemptNumber int           `json:"attempt_number"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Status        string        `json:"status"`
	Error         string        `json:"error,omitempty"`
	Duration      time.Duration `json:"duration"`
}

type Worker struct {
	ID             string           `json:"id"`
	Hostname       string           `json:"hostname"`
	SupportedTypes []domain.JobType `json:"supported_types"`
	MaxConcurrent  int              `json:"max_concurrent"`
	CurrentJobs    int              `json:"current_jobs"`
	Status         string           `json:"status"`
	LastHeartbeat  time.Time        `json:"last_heartbeat"`
	RegisteredAt   time.Time        `json:"registered_at"`
}
