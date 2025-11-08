package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eventprocessor/event-gateway/internal/api/http/server"
	"github.com/eventprocessor/event-gateway/internal/config"
	"github.com/eventprocessor/event-gateway/internal/kafka"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		logger.Fatal("Failed to initialize Kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()

	// Initialize HTTP server
	httpServer := server.New(cfg, kafkaProducer, logger)

	// Start server
	srv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: httpServer.GetRouter(),
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting Event Gateway",
			zap.String("address", cfg.Server.Address),
			zap.String("version", "1.0.0"))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Event Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Event Gateway stopped")
}
