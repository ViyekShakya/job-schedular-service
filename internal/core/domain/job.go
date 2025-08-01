package domain

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

// Job represents the core job entity
type Job struct {
	ID          uuid.UUID       `json:"id"`
	Type        JobType         `json:"type"`
	Priority    Priority        `json:"priority"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    Metadata        `json:"metadata"`
	Status      Status          `json:"status"`
	RetryPolicy RetryPolicy     `json:"retry_policy"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	FailedAt    *time.Time      `json:"failed_at,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
	WorkerID    string          `json:"worker_id,omitempty"`
}

type JobType string
type Status string
type Metadata map[string]interface{}

// Job statuses
const (
	StatusPending    Status = "pending"
	StatusScheduled  Status = "scheduled"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusDead       Status = "dead"
)

// Job types
const (
	JobTypeEmail      JobType = "email"
	JobTypePayment    JobType = "payment"
	JobTypeDataExport JobType = "data_export"
)

func (j *Job) CanRetry() bool {
	return j.RetryPolicy.AttemptsLeft() > 0 && j.Status == StatusFailed
}

func (j *Job) MarkAsProcessing(workerID string) {
	j.Status = StatusProcessing
	j.WorkerID = workerID
	j.UpdatedAt = time.Now()
	now := time.Now()
	j.ProcessedAt = &now
}

func (j *Job) MarkAsCompleted() {
	j.Status = StatusCompleted
	j.UpdatedAt = time.Now()
	now := time.Now()
	j.CompletedAt = &now
}

func (j *Job) MarkAsFailed(err error) {
	j.Status = StatusFailed
	j.LastError = err.Error()
	j.UpdatedAt = time.Now()
	now := time.Now()
	j.FailedAt = &now
	j.RetryPolicy.IncrementAttempts()
}

func (j *Job) MarkAsDead() {
	j.Status = StatusDead
	j.UpdatedAt = time.Now()
}

func (j *Job) NextRetryAt() time.Time {
	return j.RetryPolicy.NextRetryAt()
}
