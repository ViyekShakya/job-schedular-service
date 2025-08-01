package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Job processing
	DispatchInterval  time.Duration // How often to check for new jobs
	JobTimeout        time.Duration // Max time for job execution
	MaxConcurrentJobs int           // Per worker

	// Retry configuration
	DefaultMaxRetries int
	BaseRetryDelay    time.Duration
	MaxRetryDelay     time.Duration

	// Worker configuration
	WorkerHeartbeatInterval time.Duration
	WorkerTimeout           time.Duration

	// Queue configuration
	HighPriorityQueueSize   int
	MediumPriorityQueueSize int
	LowPriorityQueueSize    int
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://admin:secret@localhost/mydb?sslmode=disable"),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379"),
		DispatchInterval:        getDuration("DISPATCH_INTERVAL", "5s"),
		JobTimeout:              getDuration("JOB_TIMEOUT", "30m"),
		MaxConcurrentJobs:       getInt("MAX_CONCURRENT_JOBS", 10),
		DefaultMaxRetries:       getInt("DEFAULT_MAX_RETRIES", 3),
		BaseRetryDelay:          getDuration("BASE_RETRY_DELAY", "30s"),
		MaxRetryDelay:           getDuration("MAX_RETRY_DELAY", "30m"),
		WorkerHeartbeatInterval: getDuration("WORKER_HEARTBEAT_INTERVAL", "30s"),
		WorkerTimeout:           getDuration("WORKER_TIMEOUT", "5m"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	d, _ := time.ParseDuration(defaultValue)
	return d
}
