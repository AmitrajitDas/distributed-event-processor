package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/eventprocessor/event-gateway/internal/config"
	"github.com/eventprocessor/event-gateway/internal/models"
	"go.uber.org/zap"
)

type Producer struct {
	producer sarama.SyncProducer
	config   config.KafkaConfig
	logger   *zap.Logger
}

func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = cfg.Retries
	config.Producer.Return.Successes = true
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Flush.Messages = cfg.BatchSize

	// Use custom partitioner for better distribution
	config.Producer.Partitioner = sarama.NewHashPartitioner

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger, _ := zap.NewProduction()

	return &Producer{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

func (p *Producer) SendEvent(event *models.Event) error {
	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
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
		return fmt.Errorf("failed to send event to Kafka: %w", err)
	}

	p.logger.Debug("Event sent to Kafka",
		zap.String("event_id", event.ID),
		zap.String("topic", p.config.Topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return nil
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
