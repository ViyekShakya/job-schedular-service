// pkg/handlers/email.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"job-schedular-service/internal/core/domain"
	"math/rand"
	"time"
)

type EmailHandler struct {
	emailService EmailService // Injected dependency
}

type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	From    string `json:"from"`
}

type EmailService interface {
	SendEmail(ctx context.Context, to, subject, body, from string) error
}

func NewEmailHandler(emailService EmailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

func (h *EmailHandler) Handle(ctx context.Context, job *domain.Job) error {
	var payload EmailPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid email payload: %w", err)
	}

	// Validate payload
	if payload.To == "" || payload.Subject == "" {
		return fmt.Errorf("missing required email fields")
	}

	// Execute business logic
	return h.emailService.SendEmail(ctx, payload.To, payload.Subject, payload.Body, payload.From)
}

func (h *EmailHandler) GetType() domain.JobType {
	return domain.JobTypeEmail
}

// MockEmailService Mock implementation for development
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body, from string) error {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)

	// Simulate occasional failures (10% failure rate)
	if rand.Float32() < 0.1 {
		return fmt.Errorf("SMTP server timeout")
	}

	fmt.Printf("[EMAIL_SERVICE] Email sent to %s: %s\n", to, subject)
	return nil
}
