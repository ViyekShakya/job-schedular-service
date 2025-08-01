package logger

import (
	"context"
	"fmt"
	"job-schedular-service/internal/core/ports"
	"os"
)

type MockLogger struct{}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...ports.Field) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...ports.Field) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (m *MockLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

func (m *MockLogger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {
	fmt.Printf("[FATAL] %s %v\n", msg, fields)
	os.Exit(1)
}
