package queue

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"sync"
	"time"
)

// MemoryQueue implements QueuePort for testing/development
type MemoryQueue struct {
	mu          sync.RWMutex
	queues      map[domain.Priority][]uuid.UUID
	delayedJobs map[uuid.UUID]time.Time
	deadLetter  []uuid.UUID
	jobStorage  map[uuid.UUID]*domain.Job
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues:      make(map[domain.Priority][]uuid.UUID),
		delayedJobs: make(map[uuid.UUID]time.Time),
		deadLetter:  make([]uuid.UUID, 0),
		jobStorage:  make(map[uuid.UUID]*domain.Job),
	}
}

func (m *MemoryQueue) Enqueue(ctx context.Context, job *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store job reference
	m.jobStorage[job.ID] = job

	// Add to priority queue
	m.queues[job.Priority] = append(m.queues[job.Priority], job.ID)

	fmt.Printf("[MEMORY_QUEUE] Job %s enqueued to %s queue\n", job.ID, job.Priority.String())
	return nil
}

func (m *MemoryQueue) Dequeue(ctx context.Context, priorities []domain.Priority) (*domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check queues in priority order
	for _, priority := range priorities {
		queue := m.queues[priority]
		if len(queue) > 0 {
			// Pop first job
			jobID := queue[0]
			m.queues[priority] = queue[1:]

			job := m.jobStorage[jobID]
			if job != nil {
				fmt.Printf("[MEMORY_QUEUE] Job %s dequeued from %s queue\n", jobID, priority.String())
				return job, nil
			}
		}
	}

	return nil, fmt.Errorf("no jobs available")
}

func (m *MemoryQueue) EnqueueDelayed(ctx context.Context, job *domain.Job, delayUntil time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobStorage[job.ID] = job
	m.delayedJobs[job.ID] = delayUntil

	fmt.Printf("[MEMORY_QUEUE] Job %s scheduled for %v\n", job.ID, delayUntil)
	return nil
}

func (m *MemoryQueue) MoveToDeadLetter(ctx context.Context, job *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deadLetter = append(m.deadLetter, job.ID)
	fmt.Printf("[MEMORY_QUEUE] Job %s moved to dead letter queue\n", job.ID)
	return nil
}

func (m *MemoryQueue) GetQueueStats(ctx context.Context) (ports.QueueStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ports.QueueStats{
		ActiveJobs:     make(map[domain.Priority]int64),
		DelayedJobs:    int64(len(m.delayedJobs)),
		DeadLetterJobs: int64(len(m.deadLetter)),
	}

	for priority, queue := range m.queues {
		stats.ActiveJobs[priority] = int64(len(queue))
	}

	return stats, nil
}

func (m *MemoryQueue) ProcessDelayedJobs(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var readyJobs []uuid.UUID

	// Find jobs ready to be processed
	for jobID, scheduledTime := range m.delayedJobs {
		if scheduledTime.Before(now) || scheduledTime.Equal(now) {
			readyJobs = append(readyJobs, jobID)
		}
	}

	// Move ready jobs to active queues
	for _, jobID := range readyJobs {
		job := m.jobStorage[jobID]
		if job != nil {
			// Remove from delayed
			delete(m.delayedJobs, jobID)

			// Add to active queue
			job.Status = domain.StatusPending
			m.queues[job.Priority] = append(m.queues[job.Priority], jobID)

			fmt.Printf("[MEMORY_QUEUE] Delayed job %s moved to %s queue\n", jobID, job.Priority.String())
		}
	}

	return nil
}
