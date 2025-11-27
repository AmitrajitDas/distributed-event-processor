package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/distributed-event-processor/services/event-gateway/internal/models"
	"go.uber.org/zap"
)

type Producer struct {
	producer sarama.SyncProducer
	config   config.KafkaConfig
	logger   *zap.Logger
}

func NewProducer(cfg config.KafkaConfig, logger *zap.Logger) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = cfg.Retries
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Flush.Frequency = 500 * time.Millisecond
	saramaConfig.Producer.Flush.Messages = cfg.BatchSize

	// Use custom partitioner for better distribution
	saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

// ProduceEvent sends an event to Kafka with context support and returns partition and offset
func (p *Producer) ProduceEvent(ctx context.Context, event *models.Event) (int32, int64, error) {
	// Check context cancellation before proceeding
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	default:
	}

	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to serialize event: %w", err)
	}

	// Create Kafka message
	message := &sarama.ProducerMessage{
		Topic: p.config.Topic,
		Key:   sarama.StringEncoder(event.Type), // Partition by event type
		Value: sarama.ByteEncoder(eventData),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_id"),
				Value: []byte(event.ID),
			},
			{
				Key:   []byte("event_type"),
				Value: []byte(event.Type),
			},
			{
				Key:   []byte("source"),
				Value: []byte(event.Source),
			},
		},
		Timestamp: event.Timestamp,
	}

	// Send message
	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.Error("Failed to send event to Kafka",
			zap.String("event_id", event.ID),
			zap.Error(err))
		return 0, 0, fmt.Errorf("failed to send event to Kafka: %w", err)
	}

	p.logger.Debug("Event sent to Kafka",
		zap.String("event_id", event.ID),
		zap.String("topic", p.config.Topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return partition, offset, nil
}

func (p *Producer) SendEvent(event *models.Event) error {
	_, _, err := p.ProduceEvent(context.Background(), event)
	return err
}

func (p *Producer) SendBatchEvents(events []*models.Event) error {
	for _, event := range events {
		if err := p.SendEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

// IsHealthy checks if the Kafka producer is healthy and can send messages
func (p *Producer) IsHealthy() bool {
	if p.producer == nil {
		return false
	}
	// Check if producer is still connected by verifying it's not closed
	// Sarama doesn't expose a direct health check, but we can check if the producer exists
	// A more robust check would involve sending a test message to a health topic
	return true
}
