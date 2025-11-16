package server

import (
	"fmt"
	"net"
	"time"

	"github.com/distributed-event-processor/services/event-gateway/internal/api/grpc/handlers"
	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/distributed-event-processor/services/event-gateway/internal/kafka"
	pb "github.com/distributed-event-processor/shared/proto/events/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server
type Server struct {
	config   config.GRPCConfig
	producer *kafka.Producer
	logger   *zap.Logger
	server   *grpc.Server
}

// New creates a new gRPC server instance
func New(cfg config.GRPCConfig, producer *kafka.Producer, logger *zap.Logger) *Server {
	return &Server{
		config:   cfg,
		producer: producer,
		logger:   logger,
	}
}

// Start initializes and starts the gRPC server
func (s *Server) Start() error {
	if !s.config.Enabled {
		s.logger.Info("gRPC server is disabled")
		return nil
	}

	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Address, err)
	}

	// Configure gRPC server options
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(uint32(s.config.MaxConcurrent)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      time.Duration(s.config.ConnectionAge) * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  time.Duration(s.config.KeepAliveTime) * time.Second,
			Timeout:               time.Duration(s.config.KeepAliveMinAge) * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             time.Duration(s.config.KeepAliveMinAge) * time.Second,
			PermitWithoutStream: true,
		}),
		// Add interceptors for logging and metrics
		grpc.ChainUnaryInterceptor(
			s.loggingInterceptor(),
			s.recoveryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			s.streamLoggingInterceptor(),
			s.streamRecoveryInterceptor(),
		),
	}

	// Create gRPC server
	s.server = grpc.NewServer(opts...)

	// Register event handler
	eventHandler := handlers.NewEventHandler(s.producer, s.logger)
	pb.RegisterEventGatewayServer(s.server, eventHandler)

	// Enable reflection for grpcurl and other tools
	reflection.Register(s.server)

	s.logger.Info("Starting gRPC server",
		zap.String("address", s.config.Address),
		zap.Int("max_connections", s.config.MaxConnections),
		zap.Int("max_concurrent_streams", s.config.MaxConcurrent),
	)

	// Start serving (blocking call)
	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.server != nil {
		s.logger.Info("Stopping gRPC server...")
		s.server.GracefulStop()
		s.logger.Info("gRPC server stopped")
	}
}

// GetServer returns the underlying gRPC server (useful for testing)
func (s *Server) GetServer() *grpc.Server {
	return s.server
}
