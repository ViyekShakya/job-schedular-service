package metric

import (
	"fmt"
	"job-schedular-service/internal/core/domain"
	"time"
)

// MockMetrics Mock implementations for demo
type MockMetrics struct{}

func (m *MockMetrics) IncrementJobsScheduled(jobType domain.JobType, priority domain.Priority) {
	fmt.Printf("[METRICS] Job scheduled: %s/%s\n", jobType, priority.String())
}

func (m *MockMetrics) IncrementJobsCompleted(jobType domain.JobType, priority domain.Priority, duration time.Duration) {
	fmt.Printf("[METRICS] Job completed: %s/%s in %v\n", jobType, priority.String(), duration)
}

func (m *MockMetrics) IncrementJobsFailed(jobType domain.JobType, priority domain.Priority, reason string) {
	fmt.Printf("[METRICS] Job failed: %s/%s - %s\n", jobType, priority.String(), reason)
}

func (m *MockMetrics) IncrementJobsRetried(jobType domain.JobType, priority domain.Priority) {
	fmt.Printf("[METRICS] Job retried: %s/%s\n", jobType, priority.String())
}

func (m *MockMetrics) IncrementJobsDeadLettered(jobType domain.JobType, priority domain.Priority) {
	fmt.Printf("[METRICS] Job dead lettered: %s/%s\n", jobType, priority.String())
}

func (m *MockMetrics) SetQueueLength(priority domain.Priority, length int64) {
	fmt.Printf("[METRICS] Queue length %s: %d\n", priority.String(), length)
}

func (m *MockMetrics) SetDelayedQueueLength(length int64) {
	fmt.Printf("[METRICS] Delayed queue length: %d\n", length)
}

func (m *MockMetrics) SetDeadLetterQueueLength(length int64) {
	fmt.Printf("[METRICS] Dead letter queue length: %d\n", length)
}

func (m *MockMetrics) SetActiveWorkers(count int64) {
	fmt.Printf("[METRICS] Active workers: %d\n", count)
}

func (m *MockMetrics) IncrementWorkerHeartbeats(workerID string) {
	fmt.Printf("[METRICS] Worker heartbeat: %s\n", workerID)
}
