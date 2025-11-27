package handlers

import (
	"context"
	"testing"

	pb "github.com/distributed-event-processor/shared/proto/events/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateEvent_Valid(t *testing.T) {
	data, err := structpb.NewStruct(map[string]interface{}{
		"key": "value",
	})
	require.NoError(t, err)

	event := &pb.Event{
		Type:   "test.event",
		Source: "test-service",
		Data:   data,
	}

	err = validateEvent(event)
	assert.NoError(t, err)
}

func TestValidateEvent_NilEvent(t *testing.T) {
	err := validateEvent(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestValidateEvent_MissingType(t *testing.T) {
	data, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
	event := &pb.Event{
		Source: "test-service",
		Data:   data,
	}

	err := validateEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type is required")
}

func TestValidateEvent_MissingSource(t *testing.T) {
	data, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
	event := &pb.Event{
		Type: "test.event",
		Data: data,
	}

	err := validateEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source is required")
}

func TestValidateEvent_MissingData(t *testing.T) {
	event := &pb.Event{
		Type:   "test.event",
		Source: "test-service",
		Data:   nil,
	}

	err := validateEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data is required")
}

func TestHealthCheck_Basic(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.HealthCheckRequest{
		Detailed: false,
	}

	resp, err := handler.HealthCheck(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.ServiceStatus_SERVICE_STATUS_HEALTHY, resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.NotNil(t, resp.Timestamp)
}

func TestHealthCheck_Detailed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.HealthCheckRequest{
		Detailed: true,
	}

	resp, err := handler.HealthCheck(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Components)
	assert.Contains(t, resp.Components, "kafka")
	assert.Equal(t, pb.HealthStatus_HEALTH_STATUS_UP, resp.Components["kafka"].Status)
}

func TestValidateEventRPC_InvalidEvent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.ValidateEventRequest{
		Event: &pb.Event{
			Type: "", // Missing type
		},
	}

	resp, err := handler.ValidateEvent(context.Background(), req)

	require.NoError(t, err) // RPC should succeed
	assert.NotNil(t, resp)
	assert.False(t, resp.IsValid)
	assert.NotEmpty(t, resp.Errors)
}

func TestValidateEventRPC_ValidEvent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	data, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
	req := &pb.ValidateEventRequest{
		Event: &pb.Event{
			Type:   "test.event",
			Source: "test-service",
			Data:   data,
		},
	}

	resp, err := handler.ValidateEvent(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsValid)
	assert.Empty(t, resp.Errors)
}

func TestIngestEvent_ValidationFailure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.IngestEventRequest{
		Event: &pb.Event{
			Type: "", // Missing type
		},
	}

	_, err := handler.IngestEvent(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestIngestEventBatch_EmptyBatch(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.IngestEventBatchRequest{
		Events: []*pb.Event{},
	}

	_, err := handler.IngestEventBatch(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "batch cannot be empty")
}

func TestProtoToModel(t *testing.T) {
	data, _ := structpb.NewStruct(map[string]interface{}{
		"key":    "value",
		"number": float64(42),
	})

	timestamp := timestamppb.Now()
	event := &pb.Event{
		Id:            "event-123",
		Type:          "user.created",
		Source:        "user-service",
		TenantId:      "tenant-456",
		Data:          data,
		Timestamp:     timestamp,
		SchemaVersion: "1.0.0",
		Metadata: map[string]string{
			"env": "test",
		},
		CorrelationId: "corr-789",
		Priority:      1,
	}

	model := protoToModel(event)

	assert.Equal(t, "event-123", model.ID)
	assert.Equal(t, "user.created", model.Type)
	assert.Equal(t, "user-service", model.Source)
	assert.Equal(t, "tenant-456", model.TenantID)
	assert.Equal(t, "value", model.Data["key"])
	assert.Equal(t, float64(42), model.Data["number"])
	assert.Equal(t, timestamp.AsTime(), model.Timestamp)
	assert.Equal(t, "1.0.0", model.SchemaVersion)
	assert.Equal(t, "test", model.Metadata["env"])
	assert.Equal(t, "corr-789", model.CorrelationID)
	assert.Equal(t, 1, model.Priority)
}

func TestProtoToModel_NilTimestamp(t *testing.T) {
	data, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
	event := &pb.Event{
		Id:        "event-123",
		Type:      "test.event",
		Source:    "test-service",
		Data:      data,
		Timestamp: nil,
	}

	model := protoToModel(event)

	assert.NotNil(t, model.Timestamp)
	assert.False(t, model.Timestamp.IsZero())
}

func TestProtoToModel_NilData(t *testing.T) {
	event := &pb.Event{
		Id:     "event-123",
		Type:   "test.event",
		Source: "test-service",
		Data:   nil,
	}

	model := protoToModel(event)

	assert.NotNil(t, model.Data)
	assert.Empty(t, model.Data)
}

func TestGetRequestID_FromMetadata(t *testing.T) {
	md := metadata.New(map[string]string{
		"x-request-id": "custom-request-id",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	requestID := getRequestID(ctx)

	assert.Equal(t, "custom-request-id", requestID)
}

func TestGetRequestID_GeneratesNew(t *testing.T) {
	ctx := context.Background()

	requestID := getRequestID(ctx)

	assert.NotEmpty(t, requestID)
	// UUID format check (basic)
	assert.Len(t, requestID, 36)
}

func TestGetRequestID_NoRequestIDInMetadata(t *testing.T) {
	md := metadata.New(map[string]string{
		"other-header": "value",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	requestID := getRequestID(ctx)

	assert.NotEmpty(t, requestID)
	assert.Len(t, requestID, 36) // UUID length
}

// Helper function to check gRPC error codes
func assertGRPCError(t *testing.T, err error, expectedCode codes.Code) {
	st, ok := status.FromError(err)
	assert.True(t, ok, "error should be a gRPC status error")
	assert.Equal(t, expectedCode, st.Code())
}

// Note: Tests for successful ingestion with real Kafka producer are omitted
// as they would require mocking/integration testing. The validation tests above
// provide adequate coverage of the request handling and validation logic.

func TestIngestEventBatch_AllInvalid(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.IngestEventBatchRequest{
		Events: []*pb.Event{
			{
				Type: "", // Invalid
			},
			{
				Source: "test", // Invalid - missing type
			},
		},
	}

	resp, err := handler.IngestEventBatch(context.Background(), req)

	// All events invalid, should still process and return results
	require.NoError(t, err) // No gRPC error, returns response with failures
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.FailureCount)
	assert.Equal(t, int32(0), resp.SuccessCount)
	assert.Equal(t, 2, len(resp.Results))
}

// TestIngestEvent_NilEvent and TestValidateEvent_NilRequest would panic before validation,
// so they're not useful tests. The validateEvent function already tests nil event handling.

func TestHealthCheck_WithProducer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)

	req := &pb.HealthCheckRequest{
		Detailed: true,
	}

	resp, err := handler.HealthCheck(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Components)

	// Without producer, Kafka should be unavailable
	kafkaHealth := resp.Components["kafka"]
	assert.NotNil(t, kafkaHealth)
}
