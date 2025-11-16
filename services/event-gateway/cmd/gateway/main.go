package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "github.com/distributed-event-processor/services/event-gateway/internal/api/grpc/server"
	httpserver "github.com/distributed-event-processor/services/event-gateway/internal/api/http/server"
	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/distributed-event-processor/services/event-gateway/internal/kafka"
	"go.uber.org/zap"
)

func main() {
	// Load configuration first to determine environment
	cfg, err := config.Load()
	if err != nil {
		// If config fails to load, use a basic production logger for error reporting
		tempLogger, _ := zap.NewProduction()
		tempLogger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize logger based on environment
	logger, err := config.InitLogger(cfg.Environment)
	if err != nil {
		// Fallback to production logger if environment is invalid
		logger, _ = zap.NewProduction()
		logger.Error("Failed to initialize logger with configured environment, using production logger",
			zap.String("environment", cfg.Environment),
			zap.Error(err))
	}
	defer logger.Sync()

	logger.Info("Starting Event Gateway",
		zap.String("environment", cfg.Environment),
		zap.String("version", "1.0.0"))

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		logger.Fatal("Failed to initialize Kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()

	// Initialize HTTP server
	httpSrv := httpserver.New(cfg, kafkaProducer, logger)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: httpSrv.GetRouter(),
	}

	// Start HTTP server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.String("address", cfg.Server.Address),
			zap.String("version", "1.0.0"))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed to start", zap.Error(err))
		}
	}()

	// Initialize and start gRPC server
	grpcSrv := grpcserver.New(cfg.GRPC, kafkaProducer, logger)

	// Start gRPC server in goroutine
	grpcErrChan := make(chan error, 1)
	go func() {
		logger.Info("Starting gRPC server",
			zap.String("address", cfg.GRPC.Address),
			zap.Bool("enabled", cfg.GRPC.Enabled))

		if err := grpcSrv.Start(); err != nil {
			grpcErrChan <- err
		}
	}()

	// Wait for interrupt signal or gRPC error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Received shutdown signal")
	case err := <-grpcErrChan:
		logger.Error("gRPC server error", zap.Error(err))
	}

	logger.Info("Shutting down Event Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}

	// Shutdown gRPC server
	grpcSrv.Stop()

	logger.Info("Event Gateway stopped")
}
