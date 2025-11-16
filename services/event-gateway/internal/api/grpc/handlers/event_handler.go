package handlers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/distributed-event-processor/services/event-gateway/internal/kafka"
	"github.com/distributed-event-processor/services/event-gateway/internal/models"
	pb "github.com/distributed-event-processor/shared/proto/events/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventHandler implements the EventGateway gRPC service
type EventHandler struct {
	pb.UnimplementedEventGatewayServer
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewEventHandler creates a new gRPC event handler
func NewEventHandler(producer *kafka.Producer, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		producer: producer,
		logger:   logger,
	}
}

// IngestEvent handles single event ingestion
func (h *EventHandler) IngestEvent(ctx context.Context, req *pb.IngestEventRequest) (*pb.IngestEventResponse, error) {
	// Extract request ID from metadata or generate new one
	requestID := getRequestID(ctx)

	h.logger.Info("Received gRPC event ingestion request",
		zap.String("request_id", requestID),
		zap.String("event_type", req.Event.Type),
		zap.String("tenant_id", req.Event.TenantId),
	)

	// Validate event
	if err := validateEvent(req.Event); err != nil {
		h.logger.Warn("Event validation failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Generate event ID if not provided
	if req.Event.Id == "" {
		req.Event.Id = uuid.New().String()
	}

	// Set timestamp if not provided
	if req.Event.Timestamp == nil {
		req.Event.Timestamp = timestamppb.Now()
	}

	// Convert to internal model
	event := protoToModel(req.Event)

	// Produce to Kafka
	partition, offset, err := h.producer.ProduceEvent(ctx, event)
	if err != nil {
		h.logger.Error("Failed to produce event to Kafka",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to process event")
	}

	h.logger.Info("Event successfully ingested",
		zap.String("request_id", requestID),
		zap.String("event_id", req.Event.Id),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return &pb.IngestEventResponse{
		EventId:    req.Event.Id,
		RequestId:  requestID,
		AcceptedAt: timestamppb.Now(),
		Partition:  partition,
		Offset:     offset,
		Status:     pb.IngestionStatus_INGESTION_STATUS_ACCEPTED,
	}, nil
}

// IngestEventBatch handles batch event ingestion
func (h *EventHandler) IngestEventBatch(ctx context.Context, req *pb.IngestEventBatchRequest) (*pb.IngestEventBatchResponse, error) {
	requestID := getRequestID(ctx)
	startTime := time.Now()

	h.logger.Info("Received gRPC batch event ingestion request",
		zap.String("request_id", requestID),
		zap.Int("batch_size", len(req.Events)),
	)

	if len(req.Events) == 0 {
		return nil, status.Error(codes.InvalidArgument, "batch cannot be empty")
	}

	results := make([]*pb.IngestEventResponse, 0, len(req.Events))
	successCount := int32(0)
	failureCount := int32(0)

	for i, event := range req.Events {
		// Validate event
		if err := validateEvent(event); err != nil {
			result := &pb.IngestEventResponse{
				EventId:      event.Id,
				RequestId:    requestID,
				Status:       pb.IngestionStatus_INGESTION_STATUS_REJECTED,
				ErrorMessage: err.Error(),
			}
			results = append(results, result)
			failureCount++

			if req.FailFast {
				h.logger.Warn("Batch processing stopped due to fail-fast",
					zap.String("request_id", requestID),
					zap.Int("processed", i),
					zap.Int("total", len(req.Events)),
				)
				break
			}
			continue
		}

		// Generate event ID if not provided
		if event.Id == "" {
			event.Id = uuid.New().String()
		}

		// Set timestamp if not provided
		if event.Timestamp == nil {
			event.Timestamp = timestamppb.Now()
		}

		// Convert to internal model
		internalEvent := protoToModel(event)

		// Produce to Kafka
		partition, offset, err := h.producer.ProduceEvent(ctx, internalEvent)
		if err != nil {
			result := &pb.IngestEventResponse{
				EventId:      event.Id,
				RequestId:    requestID,
				Status:       pb.IngestionStatus_INGESTION_STATUS_FAILED,
				ErrorMessage: "failed to produce to Kafka",
			}
			results = append(results, result)
			failureCount++

			if req.FailFast {
				break
			}
			continue
		}

		result := &pb.IngestEventResponse{
			EventId:    event.Id,
			RequestId:  requestID,
			AcceptedAt: timestamppb.Now(),
			Partition:  partition,
			Offset:     offset,
			Status:     pb.IngestionStatus_INGESTION_STATUS_ACCEPTED,
		}
		results = append(results, result)
		successCount++
	}

	processingTime := time.Since(startTime).Milliseconds()

	h.logger.Info("Batch processing completed",
		zap.String("request_id", requestID),
		zap.Int32("success", successCount),
		zap.Int32("failures", failureCount),
		zap.Int64("processing_time_ms", processingTime),
	)

	return &pb.IngestEventBatchResponse{
		Results:          results,
		SuccessCount:     successCount,
		FailureCount:     failureCount,
		RequestId:        requestID,
		ProcessingTimeMs: processingTime,
	}, nil
}

// StreamEvents handles bidirectional streaming for real-time event ingestion
func (h *EventHandler) StreamEvents(stream pb.EventGateway_StreamEventsServer) error {
	requestID := uuid.New().String()
	ctx := stream.Context()

	h.logger.Info("Stream connection established",
		zap.String("request_id", requestID),
	)

	defer func() {
		h.logger.Info("Stream connection closed",
			zap.String("request_id", requestID),
		)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				h.logger.Error("Stream receive error",
					zap.String("request_id", requestID),
					zap.Error(err),
				)
				return status.Error(codes.Internal, "stream error")
			}

			switch msg := req.Message.(type) {
			case *pb.StreamEventRequest_Event:
				// Handle event ingestion
				response, err := h.handleStreamEvent(ctx, requestID, msg.Event)
				if err != nil {
					h.logger.Error("Failed to handle stream event",
						zap.String("request_id", requestID),
						zap.Error(err),
					)
					// Send error response
					statusMsg := &pb.StreamEventResponse{
						Message: &pb.StreamEventResponse_Status{
							Status: &pb.StreamStatus{
								Code:      pb.StatusCode_STATUS_CODE_ERROR,
								Message:   err.Error(),
								Timestamp: timestamppb.Now(),
							},
						},
					}
					if err := stream.Send(statusMsg); err != nil {
						return err
					}
					continue
				}

				// Send acknowledgment
				ackMsg := &pb.StreamEventResponse{
					Message: &pb.StreamEventResponse_Ack{
						Ack: response,
					},
				}
				if err := stream.Send(ackMsg); err != nil {
					return err
				}

			case *pb.StreamEventRequest_Ping:
				// Handle ping
				pongMsg := &pb.StreamEventResponse{
					Message: &pb.StreamEventResponse_Pong{
						Pong: &pb.Pong{
							Timestamp: timestamppb.Now(),
						},
					},
				}
				if err := stream.Send(pongMsg); err != nil {
					return err
				}

			case *pb.StreamEventRequest_Config:
				// Handle stream configuration
				h.logger.Info("Stream configuration received",
					zap.String("request_id", requestID),
					zap.Bool("compression", msg.Config.EnableCompression),
					zap.Int32("batch_size", msg.Config.BatchSize),
				)
			}
		}
	}
}

// ValidateEvent validates an event without persisting it
func (h *EventHandler) ValidateEvent(ctx context.Context, req *pb.ValidateEventRequest) (*pb.ValidateEventResponse, error) {
	requestID := getRequestID(ctx)

	h.logger.Info("Received validation request",
		zap.String("request_id", requestID),
		zap.String("event_type", req.Event.Type),
	)

	errors := make([]*pb.ValidationError, 0)

	// Validate event
	if err := validateEvent(req.Event); err != nil {
		errors = append(errors, &pb.ValidationError{
			Field:   "event",
			Message: err.Error(),
			Code:    "VALIDATION_FAILED",
		})
	}

	// Check required fields
	if req.Event.Type == "" {
		errors = append(errors, &pb.ValidationError{
			Field:   "type",
			Message: "event type is required",
			Code:    "REQUIRED_FIELD",
		})
	}

	if req.Event.Source == "" {
		errors = append(errors, &pb.ValidationError{
			Field:   "source",
			Message: "event source is required",
			Code:    "REQUIRED_FIELD",
		})
	}

	isValid := len(errors) == 0

	return &pb.ValidateEventResponse{
		IsValid:   isValid,
		Errors:    errors,
		RequestId: requestID,
	}, nil
}

// HealthCheck returns the health status of the gateway
func (h *EventHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	response := &pb.HealthCheckResponse{
		Status:    pb.ServiceStatus_SERVICE_STATUS_HEALTHY,
		Version:   "1.0.0",
		Timestamp: timestamppb.Now(),
	}

	if req.Detailed {
		components := make(map[string]*pb.ComponentHealth)

		// Check Kafka connectivity
		kafkaHealth := &pb.ComponentHealth{
			Status:    pb.HealthStatus_HEALTH_STATUS_UP,
			Message:   "Kafka producer is healthy",
			LastCheck: timestamppb.Now(),
		}
		components["kafka"] = kafkaHealth

		response.Components = components
	}

	return response, nil
}

// Helper functions

func (h *EventHandler) handleStreamEvent(ctx context.Context, requestID string, event *pb.Event) (*pb.IngestEventResponse, error) {
	// Validate event
	if err := validateEvent(event); err != nil {
		return nil, err
	}

	// Generate event ID if not provided
	if event.Id == "" {
		event.Id = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp == nil {
		event.Timestamp = timestamppb.Now()
	}

	// Convert to internal model
	internalEvent := protoToModel(event)

	// Produce to Kafka
	partition, offset, err := h.producer.ProduceEvent(ctx, internalEvent)
	if err != nil {
		return nil, err
	}

	return &pb.IngestEventResponse{
		EventId:    event.Id,
		RequestId:  requestID,
		AcceptedAt: timestamppb.Now(),
		Partition:  partition,
		Offset:     offset,
		Status:     pb.IngestionStatus_INGESTION_STATUS_ACCEPTED,
	}, nil
}

func validateEvent(event *pb.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}

	if event.Source == "" {
		return fmt.Errorf("event source is required")
	}

	if event.Data == nil {
		return fmt.Errorf("event data is required")
	}

	return nil
}

func protoToModel(event *pb.Event) *models.Event {
	var timestamp time.Time
	if event.Timestamp != nil {
		timestamp = event.Timestamp.AsTime()
	} else {
		timestamp = time.Now()
	}

	// Convert protobuf Struct to map
	data := make(map[string]interface{})
	if event.Data != nil {
		data = event.Data.AsMap()
	}

	return &models.Event{
		ID:            event.Id,
		Type:          event.Type,
		Source:        event.Source,
		TenantID:      event.TenantId,
		Data:          data,
		Timestamp:     timestamp,
		SchemaVersion: event.SchemaVersion,
		Metadata:      event.Metadata,
		CorrelationID: event.CorrelationId,
		Priority:      int(event.Priority),
	}
}

func getRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.New().String()
	}

	requestIDs := md.Get("x-request-id")
	if len(requestIDs) > 0 {
		return requestIDs[0]
	}

	return uuid.New().String()
}
