package domain

import (
	"job-schedular-service/internal/config"
	"math"
	"time"
)

type RetryPolicy struct {
	MaxAttempts     int           `json:"max_attempts"`
	CurrentAttempts int           `json:"current_attempts"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffStrategy BackoffType   `json:"backoff_strategy"`
	LastAttemptAt   *time.Time    `json:"last_attempt_at,omitempty"`
}

type BackoffType string

const (
	BackoffFixed       BackoffType = "fixed"
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
	DefaultBackoff     BackoffType = "exponential"
)

func NewRetryPolicy() RetryPolicy {
	cfg := config.LoadConfig()
	return RetryPolicy{
		MaxAttempts:     cfg.DefaultMaxRetries,
		CurrentAttempts: 0,
		BaseDelay:       cfg.BaseRetryDelay,
		MaxDelay:        cfg.MaxRetryDelay,
		BackoffStrategy: DefaultBackoff,
	}
}

func (rp *RetryPolicy) AttemptsLeft() int {
	return rp.MaxAttempts - rp.CurrentAttempts
}

func (rp *RetryPolicy) IncrementAttempts() {
	rp.CurrentAttempts++
	now := time.Now()
	rp.LastAttemptAt = &now
}

func (rp *RetryPolicy) NextRetryAt() time.Time {
	if rp.LastAttemptAt == nil {
		return time.Now()
	}

	var delay time.Duration

	switch rp.BackoffStrategy {
	case BackoffFixed:
		delay = rp.BaseDelay
	case BackoffLinear:
		delay = time.Duration(rp.CurrentAttempts) * rp.BaseDelay
	case BackoffExponential:
		delay = time.Duration(math.Pow(2, float64(rp.CurrentAttempts))) * rp.BaseDelay
	default:
		delay = rp.BaseDelay
	}

	if delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	return rp.LastAttemptAt.Add(delay)
}
