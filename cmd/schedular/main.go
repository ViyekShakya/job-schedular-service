package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"job-schedular-service/internal/adapters/primary/rest_api"
	"job-schedular-service/internal/adapters/secondary/logger"
	"job-schedular-service/internal/adapters/secondary/metric"
	"job-schedular-service/internal/adapters/secondary/queue"
	"job-schedular-service/internal/adapters/secondary/storage"
	"job-schedular-service/internal/config"
	services2 "job-schedular-service/internal/core/services"
	"job-schedular-service/pkg/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Initialize dependencies (using memory adapters for demo)
	cfg := config.LoadConfig()
	storageAdapter, _ := storage.NewPostgresStorage(cfg.DatabaseURL)
	queueAdapter := queue.NewRedisQueue(cfg.RedisURL)

	// Mock adapters for metrics and logging
	metricsAdapter := &metric.MockMetrics{}
	loggerAdapter := &logger.MockLogger{}

	// Initialize core services
	schedulerService := services2.NewSchedulerService(storageAdapter, queueAdapter, metricsAdapter, loggerAdapter)
	processorService := services2.NewProcessorService(storageAdapter, queueAdapter, metricsAdapter, loggerAdapter)

	// Register job handlers
	emailService := &handlers.MockEmailService{}
	paymentService := &handlers.MockPaymentService{}

	processorService.RegisterHandler(handlers.NewEmailHandler(emailService))
	processorService.RegisterHandler(handlers.NewPaymentHandler(paymentService))

	// Initialize HTTP handler
	httpHandler := rest_api.NewHandler(schedulerService, processorService)

	// Setup router
	router := gin.Default()
	api := router.Group("/api/v1")
	{
		api.POST("/jobs", httpHandler.ScheduleJob)
		api.GET("/jobs/:id", httpHandler.GetJob)
		api.GET("/jobs", httpHandler.ListJobs)
		api.GET("/queues/stats", httpHandler.GetQueueStats)
	}

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	// Don't defer cancel() here - we need to control when it's called

	var wg sync.WaitGroup

	// Start delayed job processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				queueAdapter.ProcessDelayedJobs(ctx)
			}
		}
	}()

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("Job scheduler started on :8000")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Graceful shutdown - cancel background services first
	cancel() // Cancel background services immediately

	// Then shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	log.Println("Shutdown complete")
}
