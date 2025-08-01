// internal/adapters/secondary/queue/redis.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"log"
	"time"
)

// RedisQueue implements QueuePort using Redis
type RedisQueue struct {
	client *redis.Client
}

func InitRedis(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}
func NewRedisQueue(redisURL string) *RedisQueue {
	client, err := InitRedis(redisURL)
	if err != nil {
		log.Println("Redis initialization failed: ", err)
	}
	return &RedisQueue{client: client}
}

func (r *RedisQueue) Enqueue(ctx context.Context, job *domain.Job) error {
	queueName := r.getQueueName(job.Priority)

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Use LPUSH for FIFO behavior with BRPOP
	return r.client.LPush(ctx, queueName, jobData).Err()
}

func (r *RedisQueue) Dequeue(ctx context.Context, priorities []domain.Priority) (*domain.Job, error) {
	queueNames := make([]string, len(priorities))
	for i, priority := range priorities {
		queueNames[i] = r.getQueueName(priority)
	}

	// BRPOP blocks until a job is available or timeout
	result, err := r.client.BRPop(ctx, 1*time.Second, queueNames...).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("no jobs available")
		}
		return nil, err
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid redis response")
	}

	jobData := result[1]
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (r *RedisQueue) EnqueueDelayed(ctx context.Context, job *domain.Job, delayUntil time.Time) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	score := float64(delayUntil.Unix())
	return r.client.ZAdd(ctx, "delayed_jobs", &redis.Z{
		Score:  score,
		Member: jobData,
	}).Err()
}

func (r *RedisQueue) MoveToDeadLetter(ctx context.Context, job *domain.Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return r.client.LPush(ctx, "dead_letter_queue", jobData).Err()
}

func (r *RedisQueue) ProcessDelayedJobs(ctx context.Context) error {
	now := float64(time.Now().Unix())

	// Get jobs ready to be processed
	jobs, err := r.client.ZRangeByScore(ctx, "delayed_jobs", &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%f", now),
	}).Result()

	if err != nil {
		return err
	}

	for _, jobData := range jobs {
		var job domain.Job
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			continue // Skip malformed jobs
		}

		// Move to active queue
		if err := r.Enqueue(ctx, &job); err != nil {
			continue // Skip failed enqueues
		}

		// Remove from delayed queue
		r.client.ZRem(ctx, "delayed_jobs", jobData)
	}

	return nil
}

func (r *RedisQueue) GetQueueStats(ctx context.Context) (ports.QueueStats, error) {
	stats := ports.QueueStats{
		ActiveJobs: make(map[domain.Priority]int64),
	}

	// Get active queue lengths
	priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityCritical}
	for _, priority := range priorities {
		queueName := r.getQueueName(priority)
		length, err := r.client.LLen(ctx, queueName).Result()
		if err != nil {
			return stats, err
		}
		stats.ActiveJobs[priority] = length
	}

	// Get delayed jobs count
	delayedCount, err := r.client.ZCard(ctx, "delayed_jobs").Result()
	if err != nil {
		return stats, err
	}
	stats.DelayedJobs = delayedCount

	// Get dead letter queue count
	deadCount, err := r.client.LLen(ctx, "dead_letter_queue").Result()
	if err != nil {
		return stats, err
	}
	stats.DeadLetterJobs = deadCount

	return stats, nil
}

func (r *RedisQueue) getQueueName(priority domain.Priority) string {
	return fmt.Sprintf("jobs_%s", priority.String())
}
