package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/distributed-event-processor/services/stream-processor/internal/config"
	"github.com/distributed-event-processor/services/stream-processor/internal/models"
	"go.uber.org/zap"
)

// EventHandler is a function type for handling consumed events
type EventHandler func(event *models.Event, partition int32, offset int64) error

// Consumer represents a Kafka consumer
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	config        config.KafkaConfig
	logger        *zap.Logger
	handler       EventHandler
	ready         chan bool
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig, logger *zap.Logger) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = cfg.AutoCommit
	saramaConfig.Consumer.Return.Errors = true

	// Set initial offset
	if cfg.StartOffset == "earliest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		consumerGroup: consumerGroup,
		config:        cfg,
		logger:        logger,
		ready:         make(chan bool),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start(handler EventHandler) error {
	c.handler = handler

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			// Check if context is cancelled
			if c.ctx.Err() != nil {
				return
			}

			// Create consumer group handler
			groupHandler := &consumerGroupHandler{
				consumer: c,
			}

			// Consume messages
			if err := c.consumerGroup.Consume(c.ctx, []string{c.config.Topic}, groupHandler); err != nil {
				c.logger.Error("Error from consumer", zap.Error(err))
				return
			}

			// Check if context was cancelled
			if c.ctx.Err() != nil {
				return
			}
		}
	}()

	// Start error handler
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for err := range c.consumerGroup.Errors() {
			c.logger.Error("Consumer group error", zap.Error(err))
		}
	}()

	// Wait until consumer is ready
	<-c.ready

	c.logger.Info("Kafka consumer started",
		zap.String("topic", c.config.Topic),
		zap.String("consumer_group", c.config.ConsumerGroup))

	return nil
}

// Stop stops the consumer gracefully
func (c *Consumer) Stop() error {
	c.logger.Info("Stopping Kafka consumer...")

	// Cancel context to stop consumption
	c.cancel()

	// Wait for goroutines to finish
	c.wg.Wait()

	// Close consumer group
	if err := c.consumerGroup.Close(); err != nil {
		c.logger.Error("Error closing consumer group", zap.Error(err))
		return err
	}

	c.logger.Info("Kafka consumer stopped")
	return nil
}

// Commit commits the current offsets
func (c *Consumer) Commit(session sarama.ConsumerGroupSession) error {
	session.Commit()
	return nil
}

// IsHealthy checks if the consumer is healthy
func (c *Consumer) IsHealthy() bool {
	// Check if consumer group is not closed
	select {
	case <-c.ctx.Done():
		return false
	default:
		return true
	}
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	consumer *Consumer
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.consumer.logger.Info("Consumer group session started",
		zap.String("member_id", session.MemberID()),
		zap.Int32("generation_id", session.GenerationID()))

	// Signal that consumer is ready
	close(h.consumer.ready)
	h.consumer.ready = make(chan bool)

	return nil
}

// Cleanup is run at the end of session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.consumer.logger.Info("Consumer group session ended",
		zap.String("member_id", session.MemberID()))
	return nil
}

// ConsumeClaim processes messages from a partition
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.consumer.logger.Info("Starting to consume partition",
		zap.Int32("partition", claim.Partition()),
		zap.Int64("initial_offset", claim.InitialOffset()))

	// Process messages
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Parse event from message
			var event models.Event
			if err := json.Unmarshal(message.Value, &event); err != nil {
				h.consumer.logger.Error("Failed to unmarshal event",
					zap.Error(err),
					zap.Int32("partition", message.Partition),
					zap.Int64("offset", message.Offset))
				// Mark message as processed even if unmarshaling fails
				session.MarkMessage(message, "")
				continue
			}

			// Call event handler
			if err := h.consumer.handler(&event, message.Partition, message.Offset); err != nil {
				h.consumer.logger.Error("Error handling event",
					zap.Error(err),
					zap.String("event_id", event.ID),
					zap.Int32("partition", message.Partition),
					zap.Int64("offset", message.Offset))
				// Continue processing even if handler fails
				// The handler should implement its own error handling/retry logic
			}

			// Mark message as processed (manual commit mode)
			session.MarkMessage(message, "")

			h.consumer.logger.Debug("Processed message",
				zap.String("event_id", event.ID),
				zap.String("event_type", event.Type),
				zap.Int32("partition", message.Partition),
				zap.Int64("offset", message.Offset))

		case <-session.Context().Done():
			return nil
		}
	}
}
