package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/distributed-event-processor/services/event-gateway/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func createTestProducer(t *testing.T, mockProducer *mocks.SyncProducer) *Producer {
	logger, _ := zap.NewDevelopment()
	cfg := config.KafkaConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     "test-events",
		Retries:   3,
		BatchSize: 100,
	}

	return &Producer{
		producer: mockProducer,
		config:   cfg,
		logger:   logger,
	}
}

func createTestEvent() *models.Event {
	return &models.Event{
		ID:        "test-event-123",
		Type:      "user.created",
		Source:    "test-service",
		Subject:   "user-456",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"env": "test"},
	}
}

func TestProduceEvent_Success(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()

	producer := createTestProducer(t, mockProducer)
	event := createTestEvent()

	partition, offset, err := producer.ProduceEvent(context.Background(), event)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, partition, int32(0))
	assert.GreaterOrEqual(t, offset, int64(0))
}

func TestProduceEvent_Failure(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndFail(sarama.ErrNotLeaderForPartition)

	producer := createTestProducer(t, mockProducer)
	event := createTestEvent()

	_, _, err := producer.ProduceEvent(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send event to Kafka")
}

func TestProduceEvent_ContextCancelled(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	producer := createTestProducer(t, mockProducer)
	event := createTestEvent()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := producer.ProduceEvent(ctx, event)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestSendEvent_Success(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()

	producer := createTestProducer(t, mockProducer)
	event := createTestEvent()

	err := producer.SendEvent(event)

	require.NoError(t, err)
}

func TestSendEvent_Failure(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndFail(sarama.ErrBrokerNotAvailable)

	producer := createTestProducer(t, mockProducer)
	event := createTestEvent()

	err := producer.SendEvent(event)

	require.Error(t, err)
}

func TestSendBatchEvents_Success(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	// Expect 3 messages
	mockProducer.ExpectSendMessageAndSucceed()
	mockProducer.ExpectSendMessageAndSucceed()
	mockProducer.ExpectSendMessageAndSucceed()

	producer := createTestProducer(t, mockProducer)
	events := []*models.Event{
		createTestEvent(),
		createTestEvent(),
		createTestEvent(),
	}

	err := producer.SendBatchEvents(events)

	require.NoError(t, err)
}

func TestSendBatchEvents_PartialFailure(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	mockProducer.ExpectSendMessageAndFail(sarama.ErrNotLeaderForPartition)

	producer := createTestProducer(t, mockProducer)
	events := []*models.Event{
		createTestEvent(),
		createTestEvent(),
	}

	err := producer.SendBatchEvents(events)

	require.Error(t, err)
}

func TestSendBatchEvents_EmptyBatch(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	producer := createTestProducer(t, mockProducer)

	err := producer.SendBatchEvents([]*models.Event{})

	require.NoError(t, err)
}

func TestIsHealthy(t *testing.T) {
	t.Run("healthy when producer exists", func(t *testing.T) {
		mockProducer := mocks.NewSyncProducer(t, nil)
		producer := createTestProducer(t, mockProducer)

		assert.True(t, producer.IsHealthy())
	})

	t.Run("unhealthy when producer is nil", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		producer := &Producer{
			producer: nil,
			logger:   logger,
		}

		assert.False(t, producer.IsHealthy())
	})
}

func TestClose(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	producer := createTestProducer(t, mockProducer)

	err := producer.Close()

	require.NoError(t, err)
}
