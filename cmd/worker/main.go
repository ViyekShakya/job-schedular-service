// cmd/worker/main.go
package main

import (
	"context"
	"job-schedular-service/internal/adapters/primary/worker"
	"job-schedular-service/internal/adapters/secondary/logger"
	"job-schedular-service/internal/adapters/secondary/metric"
	"job-schedular-service/internal/adapters/secondary/queue"
	"job-schedular-service/internal/adapters/secondary/storage"
	"job-schedular-service/internal/config"
	"job-schedular-service/internal/core/services"
	"job-schedular-service/pkg/handlers"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Initialize dependencies (should use same adapters as scheduler)
	cfg := config.LoadConfig()
	storageAdapter, _ := storage.NewPostgresStorage(cfg.DatabaseURL)
	queueAdapter := queue.NewRedisQueue(cfg.RedisURL)
	metricsAdapter := &metric.MockMetrics{}
	loggerAdapter := &logger.MockLogger{}

	// Initialize processor service
	processorService := services.NewProcessorService(storageAdapter, queueAdapter, metricsAdapter, loggerAdapter)

	// Register job handlers
	emailService := &handlers.MockEmailService{}
	paymentService := &handlers.MockPaymentService{}

	processorService.RegisterHandler(handlers.NewEmailHandler(emailService))
	processorService.RegisterHandler(handlers.NewPaymentHandler(paymentService))

	// Initialize worker runner
	workerRunner := worker.NewWorkerRunner(
		processorService,
		storageAdapter,
		loggerAdapter,
		"worker-1",
		"localhost",
		5, // max concurrent jobs
	)

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := workerRunner.Start(ctx); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	log.Println("Worker started")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Worker shutting down...")
	cancel()
	log.Println("Worker stopped")
}
