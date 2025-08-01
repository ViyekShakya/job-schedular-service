package storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"sync"
	"time"
)

// MemoryStorage implements StoragePort for testing/development
type MemoryStorage struct {
	mu      sync.RWMutex
	jobs    map[uuid.UUID]*domain.Job
	history map[uuid.UUID][]*ports.JobHistory
	workers map[string]*ports.Worker
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		jobs:    make(map[uuid.UUID]*domain.Job),
		history: make(map[uuid.UUID][]*ports.JobHistory),
		workers: make(map[string]*ports.Worker),
	}
}

func (m *MemoryStorage) SaveJob(ctx context.Context, job *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobs[job.ID] = job
	fmt.Printf("[MEMORY_STORAGE] Job %s saved\n", job.ID)
	return nil
}

func (m *MemoryStorage) GetJob(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	return job, nil
}

func (m *MemoryStorage) UpdateJob(ctx context.Context, job *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.ID]; !exists {
		return fmt.Errorf("job not found")
	}

	job.UpdatedAt = time.Now()
	m.jobs[job.ID] = job
	fmt.Printf("[MEMORY_STORAGE] Job %s updated (status: %s)\n", job.ID, job.Status)
	return nil
}

func (m *MemoryStorage) ListJobs(ctx context.Context, filter ports.JobFilter) ([]*domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*domain.Job
	count := 0
	skipped := 0

	for _, job := range m.jobs {
		// Apply filters
		if filter.Status != nil && job.Status != *filter.Status {
			continue
		}
		if filter.JobType != nil && job.Type != *filter.JobType {
			continue
		}
		if filter.Priority != nil && job.Priority != *filter.Priority {
			continue
		}

		// Apply offset
		if skipped < filter.Offset {
			skipped++
			continue
		}

		// Apply limit
		if filter.Limit > 0 && count >= filter.Limit {
			break
		}

		jobs = append(jobs, job)
		count++
	}

	return jobs, nil
}

func (m *MemoryStorage) SaveJobHistory(ctx context.Context, history *ports.JobHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history[history.JobID] = append(m.history[history.JobID], history)
	fmt.Printf("[MEMORY_STORAGE] Job history saved for job %s (attempt %d)\n", history.JobID, history.AttemptNumber)
	return nil
}

func (m *MemoryStorage) GetJobHistory(ctx context.Context, jobID uuid.UUID) ([]*ports.JobHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.history[jobID]
	if !exists {
		return []*ports.JobHistory{}, nil
	}

	return history, nil
}

func (m *MemoryStorage) RegisterWorker(ctx context.Context, worker *ports.Worker) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workers[worker.ID] = worker
	fmt.Printf("[MEMORY_STORAGE] Worker %s registered\n", worker.ID)
	return nil
}

func (m *MemoryStorage) UpdateWorkerHeartbeat(ctx context.Context, workerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	worker, exists := m.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found")
	}

	worker.LastHeartbeat = time.Now()
	fmt.Printf("[MEMORY_STORAGE] Worker %s heartbeat updated\n", workerID)
	return nil
}

func (m *MemoryStorage) GetActiveWorkers(ctx context.Context) ([]*ports.Worker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var activeWorkers []*ports.Worker
	for _, worker := range m.workers {
		if worker.Status == "active" {
			activeWorkers = append(activeWorkers, worker)
		}
	}

	return activeWorkers, nil
}

func (m *MemoryStorage) DeactivateStaleWorkers(ctx context.Context, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	deactivated := 0

	for _, worker := range m.workers {
		if worker.LastHeartbeat.Before(cutoff) && worker.Status == "active" {
			worker.Status = "inactive"
			deactivated++
		}
	}

	if deactivated > 0 {
		fmt.Printf("[MEMORY_STORAGE] Deactivated %d stale workers\n", deactivated)
	}

	return nil
}
