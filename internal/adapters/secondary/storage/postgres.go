package storage

import (
	"context"
	"encoding/json"
	"fmt"
	domain2 "job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GORM Models
type JobModel struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Type        string           `gorm:"size:50;not null;index"`
	Priority    domain2.Priority `gorm:"size:20;not null;index"`
	Payload     string           `gorm:"type:jsonb;not null"`
	Metadata    string           `gorm:"type:jsonb"`
	Status      string           `gorm:"size:20;not null;index"`
	RetryPolicy string           `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time        `gorm:"not null"`
	UpdatedAt   time.Time        `gorm:"not null"`
	ScheduledAt time.Time        `gorm:"not null;index"`
	ProcessedAt *time.Time
	CompletedAt *time.Time
	FailedAt    *time.Time
	LastError   string `gorm:"type:text"`
	WorkerID    string `gorm:"size:100"`

	History []JobHistoryModel `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
}

func (JobModel) TableName() string {
	return "jobs"
}

type JobHistoryModel struct {
	ID            string    `gorm:"size:100;primaryKey"`
	JobID         uuid.UUID `gorm:"type:uuid;not null;index"`
	WorkerID      string    `gorm:"size:100;not null"`
	AttemptNumber int       `gorm:"not null"`
	StartedAt     time.Time `gorm:"not null"`
	CompletedAt   *time.Time
	Status        string `gorm:"size:20;not null"`
	ErrorMessage  string `gorm:"type:text"`
	Duration      int64  `gorm:"not null;default:0"` // nanoseconds
}

func (JobHistoryModel) TableName() string {
	return "job_history"
}

type WorkerModel struct {
	ID             string    `gorm:"size:100;primaryKey"`
	Hostname       string    `gorm:"size:255;not null"`
	SupportedTypes string    `gorm:"type:text;not null"` // JSON array
	MaxConcurrent  int       `gorm:"not null"`
	CurrentJobs    int       `gorm:"not null;default:0"`
	Status         string    `gorm:"size:20;not null;index"`
	LastHeartbeat  time.Time `gorm:"not null;index"`
	RegisteredAt   time.Time `gorm:"not null"`
}

func (WorkerModel) TableName() string {
	return "workers"
}

type PostgresStorage struct {
	db *gorm.DB
}

func NewPostgresStorage(databaseURL string) (*PostgresStorage, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &PostgresStorage{db: db}

	// Auto-migrate the schema
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return storage, nil
}

func (p *PostgresStorage) migrate() error {
	return p.db.AutoMigrate(&JobModel{}, &JobHistoryModel{}, &WorkerModel{})
}

func (p *PostgresStorage) SaveJob(ctx context.Context, job *domain2.Job) error {
	jobModel, err := p.domainToJobModel(job)
	if err != nil {
		return fmt.Errorf("failed to convert job to model: %w", err)
	}

	// Use GORM's Save method which handles both insert and update
	result := p.db.WithContext(ctx).Save(jobModel)
	return result.Error
}

func (p *PostgresStorage) GetJob(ctx context.Context, id uuid.UUID) (*domain2.Job, error) {
	var jobModel JobModel
	result := p.db.WithContext(ctx).Where("id = ?", id).First(&jobModel)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found")
		}
		return nil, result.Error
	}

	return p.jobModelToDomain(&jobModel)
}

func (p *PostgresStorage) UpdateJob(ctx context.Context, job *domain2.Job) error {
	jobModel, err := p.domainToJobModel(job)
	if err != nil {
		return fmt.Errorf("failed to convert job to model: %w", err)
	}

	result := p.db.WithContext(ctx).Save(jobModel)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job with id %s not found", job.ID)
	}

	return nil
}

func (p *PostgresStorage) ListJobs(ctx context.Context, filter ports.JobFilter) ([]*domain2.Job, error) {
	query := p.db.WithContext(ctx).Model(&JobModel{})

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.JobType != nil {
		query = query.Where("type = ?", *filter.JobType)
	}

	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}

	query = query.Order("created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var jobModels []JobModel
	result := query.Find(&jobModels)
	if result.Error != nil {
		return nil, result.Error
	}

	jobs := make([]*domain2.Job, len(jobModels))
	for i, jobModel := range jobModels {
		job, err := p.jobModelToDomain(&jobModel)
		if err != nil {
			return nil, err
		}
		jobs[i] = job
	}

	return jobs, nil
}

func (p *PostgresStorage) SaveJobHistory(ctx context.Context, history *ports.JobHistory) error {
	historyModel := &JobHistoryModel{
		ID:            history.ID,
		JobID:         history.JobID,
		WorkerID:      history.WorkerID,
		AttemptNumber: history.AttemptNumber,
		StartedAt:     history.StartedAt,
		CompletedAt:   history.CompletedAt,
		Status:        history.Status,
		ErrorMessage:  history.Error,
		Duration:      int64(history.Duration),
	}

	result := p.db.WithContext(ctx).Save(historyModel)
	return result.Error
}

func (p *PostgresStorage) GetJobHistory(ctx context.Context, jobID uuid.UUID) ([]*ports.JobHistory, error) {
	var historyModels []JobHistoryModel
	result := p.db.WithContext(ctx).Where("job_id = ?", jobID).Order("started_at DESC").Find(&historyModels)

	if result.Error != nil {
		return nil, result.Error
	}

	histories := make([]*ports.JobHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = &ports.JobHistory{
			ID:            model.ID,
			JobID:         model.JobID,
			WorkerID:      model.WorkerID,
			AttemptNumber: model.AttemptNumber,
			StartedAt:     model.StartedAt,
			CompletedAt:   model.CompletedAt,
			Status:        model.Status,
			Error:         model.ErrorMessage,
			Duration:      time.Duration(model.Duration),
		}
	}

	return histories, nil
}

func (p *PostgresStorage) RegisterWorker(ctx context.Context, worker *ports.Worker) error {
	supportedTypesJSON, err := json.Marshal(worker.SupportedTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal supported types: %w", err)
	}

	workerModel := &WorkerModel{
		ID:             worker.ID,
		Hostname:       worker.Hostname,
		SupportedTypes: string(supportedTypesJSON),
		MaxConcurrent:  worker.MaxConcurrent,
		CurrentJobs:    worker.CurrentJobs,
		Status:         worker.Status,
		LastHeartbeat:  worker.LastHeartbeat,
		RegisteredAt:   worker.RegisteredAt,
	}

	result := p.db.WithContext(ctx).Save(workerModel)
	return result.Error
}

func (p *PostgresStorage) UpdateWorkerHeartbeat(ctx context.Context, workerID string) error {
	result := p.db.WithContext(ctx).Model(&WorkerModel{}).
		Where("id = ?", workerID).
		Update("last_heartbeat", time.Now())

	return result.Error
}

func (p *PostgresStorage) GetActiveWorkers(ctx context.Context) ([]*ports.Worker, error) {
	var workerModels []WorkerModel
	result := p.db.WithContext(ctx).Where("status = ?", "active").Find(&workerModels)

	if result.Error != nil {
		return nil, result.Error
	}

	workers := make([]*ports.Worker, len(workerModels))
	for i, model := range workerModels {
		var supportedTypes []domain2.JobType
		if err := json.Unmarshal([]byte(model.SupportedTypes), &supportedTypes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal supported types: %w", err)
		}

		workers[i] = &ports.Worker{
			ID:             model.ID,
			Hostname:       model.Hostname,
			SupportedTypes: supportedTypes,
			MaxConcurrent:  model.MaxConcurrent,
			CurrentJobs:    model.CurrentJobs,
			Status:         model.Status,
			LastHeartbeat:  model.LastHeartbeat,
			RegisteredAt:   model.RegisteredAt,
		}
	}

	return workers, nil
}

func (p *PostgresStorage) DeactivateStaleWorkers(ctx context.Context, timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout)
	result := p.db.WithContext(ctx).Model(&WorkerModel{}).
		Where("last_heartbeat < ? AND status = ?", cutoff, "active").
		Update("status", "inactive")

	return result.Error
}

func (p *PostgresStorage) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Helper methods for conversion between domain and GORM models

func (p *PostgresStorage) domainToJobModel(job *domain2.Job) (*JobModel, error) {
	// Convert json.RawMessage to string for JSONB storage
	payloadStr := string(job.Payload)
	if payloadStr == "" || payloadStr == "null" {
		payloadStr = "{}"
	}

	metadataJSON, err := json.Marshal(job.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	retryPolicyJSON, err := json.Marshal(job.RetryPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal retry policy: %w", err)
	}

	return &JobModel{
		ID:          job.ID,
		Type:        string(job.Type),
		Priority:    job.Priority,
		Payload:     payloadStr,
		Metadata:    string(metadataJSON),
		Status:      string(job.Status),
		RetryPolicy: string(retryPolicyJSON),
		CreatedAt:   job.CreatedAt,
		UpdatedAt:   job.UpdatedAt,
		ScheduledAt: job.ScheduledAt,
		ProcessedAt: job.ProcessedAt,
		CompletedAt: job.CompletedAt,
		FailedAt:    job.FailedAt,
		LastError:   job.LastError,
		WorkerID:    job.WorkerID,
	}, nil
}

func (p *PostgresStorage) jobModelToDomain(model *JobModel) (*domain2.Job, error) {
	// Convert string back to json.RawMessage
	var payload json.RawMessage
	if model.Payload != "" {
		payload = json.RawMessage(model.Payload)
	}

	var metadata domain2.Metadata
	if model.Metadata != "" {
		if err := json.Unmarshal([]byte(model.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	var retryPolicy domain2.RetryPolicy
	if model.RetryPolicy != "" {
		if err := json.Unmarshal([]byte(model.RetryPolicy), &retryPolicy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal retry policy: %w", err)
		}
	}

	return &domain2.Job{
		ID:          model.ID,
		Type:        domain2.JobType(model.Type),
		Priority:    model.Priority,
		Payload:     payload,
		Metadata:    metadata,
		Status:      domain2.Status(model.Status),
		RetryPolicy: retryPolicy,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		ScheduledAt: model.ScheduledAt,
		ProcessedAt: model.ProcessedAt,
		CompletedAt: model.CompletedAt,
		FailedAt:    model.FailedAt,
		LastError:   model.LastError,
		WorkerID:    model.WorkerID,
	}, nil
}

// Additional helper methods for common operations

// GetJobsByStatus retrieves jobs by status with pagination
func (p *PostgresStorage) GetJobsByStatus(ctx context.Context, status domain2.Status, limit, offset int) ([]*domain2.Job, error) {
	var jobModels []JobModel
	result := p.db.WithContext(ctx).
		Where("status = ?", string(status)).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobModels)

	if result.Error != nil {
		return nil, result.Error
	}

	jobs := make([]*domain2.Job, len(jobModels))
	for i, jobModel := range jobModels {
		job, err := p.jobModelToDomain(&jobModel)
		if err != nil {
			return nil, err
		}
		jobs[i] = job
	}

	return jobs, nil
}

// GetPendingJobs retrieves jobs that are ready to be processed
func (p *PostgresStorage) GetPendingJobs(ctx context.Context, limit int) ([]*domain2.Job, error) {
	var jobModels []JobModel
	result := p.db.WithContext(ctx).
		Where("status IN ? AND scheduled_at <= ?",
			[]string{string(domain2.StatusPending), string(domain2.StatusScheduled)},
			time.Now()).
		Order("priority DESC, scheduled_at ASC").
		Limit(limit).
		Find(&jobModels)

	if result.Error != nil {
		return nil, result.Error
	}

	jobs := make([]*domain2.Job, len(jobModels))
	for i, jobModel := range jobModels {
		job, err := p.jobModelToDomain(&jobModel)
		if err != nil {
			return nil, err
		}
		jobs[i] = job
	}

	return jobs, nil
}

// UpdateJobStatus updates only the job status and updated_at timestamp
func (p *PostgresStorage) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status domain2.Status) error {
	result := p.db.WithContext(ctx).Model(&JobModel{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job with id %s not found", jobID)
	}

	return nil
}

// BatchInsertJobs inserts multiple jobs efficiently
func (p *PostgresStorage) BatchInsertJobs(ctx context.Context, jobs []*domain2.Job) error {
	jobModels := make([]*JobModel, len(jobs))

	for i, job := range jobs {
		jobModel, err := p.domainToJobModel(job)
		if err != nil {
			return fmt.Errorf("failed to convert job %s to model: %w", job.ID, err)
		}
		jobModels[i] = jobModel
	}

	// Use GORM's CreateInBatches for efficient bulk insert
	return p.db.WithContext(ctx).CreateInBatches(jobModels, 100).Error
}
