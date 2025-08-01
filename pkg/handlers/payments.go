package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"job-schedular-service/internal/core/domain"
	"math/rand"
	"time"
)

type PaymentHandler struct {
	paymentService PaymentService
}

type PaymentPayload struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	UserID    string  `json:"user_id"`
}

type PaymentService interface {
	ProcessPayment(ctx context.Context, paymentID string, amount float64, currency, userID string) error
}

func NewPaymentHandler(paymentService PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

func (h *PaymentHandler) Handle(ctx context.Context, job *domain.Job) error {
	var payload PaymentPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid payment payload: %w", err)
	}

	// Validate payload
	if payload.PaymentID == "" || payload.Amount <= 0 || payload.UserID == "" {
		return fmt.Errorf("invalid payment data")
	}

	return h.paymentService.ProcessPayment(ctx, payload.PaymentID, payload.Amount, payload.Currency, payload.UserID)
}

func (h *PaymentHandler) GetType() domain.JobType {
	return domain.JobTypePayment
}

// MockPaymentService Mock implementation
type MockPaymentService struct{}

func (m *MockPaymentService) ProcessPayment(ctx context.Context, paymentID string, amount float64, currency, userID string) error {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(5)+2) * time.Second)

	// Simulate failures (8% failure rate)
	if rand.Float32() < 0.08 {
		return fmt.Errorf("insufficient funds")
	}

	fmt.Printf("[PAYMENT_SERVICE] Payment %s processed: %.2f %s for user %s\n", paymentID, amount, currency, userID)
	return nil
}
