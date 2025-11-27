package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetViper() {
	viper.Reset()
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()

	cfg, err := Load()

	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Check default values
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, ":8090", cfg.Server.Address)
	assert.Equal(t, 30, cfg.Server.ReadTimeout)
	assert.Equal(t, 30, cfg.Server.WriteTimeout)
	assert.Equal(t, 120, cfg.Server.IdleTimeout)

	// Check gRPC defaults
	assert.True(t, cfg.GRPC.Enabled)
	assert.Equal(t, ":9090", cfg.GRPC.Address)
	assert.Equal(t, 10000, cfg.GRPC.MaxConnections)

	// Check WebSocket defaults
	assert.False(t, cfg.WebSocket.Enabled)
	assert.Equal(t, "/ws", cfg.WebSocket.Path)

	// Check Kafka defaults
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "events", cfg.Kafka.Topic)
	assert.Equal(t, 3, cfg.Kafka.Retries)
	assert.Equal(t, 100, cfg.Kafka.BatchSize)

	// Check metrics defaults
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)

	// Check rate limit defaults
	assert.Equal(t, 1000, cfg.RateLimit.RequestsPerSecond)
	assert.Equal(t, 2000, cfg.RateLimit.BurstSize)
}

func TestLoad_EnvironmentVariableOverride(t *testing.T) {
	resetViper()

	// Set environment variables
	os.Setenv("GATEWAY_ENVIRONMENT", "production")
	os.Setenv("GATEWAY_SERVER_ADDRESS", ":9000")
	defer func() {
		os.Unsetenv("GATEWAY_ENVIRONMENT")
		os.Unsetenv("GATEWAY_SERVER_ADDRESS")
	}()

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, ":9000", cfg.Server.Address)
}

func TestInitLogger_Development(t *testing.T) {
	logger, err := InitLogger("development")

	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestInitLogger_Dev(t *testing.T) {
	logger, err := InitLogger("dev")

	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestInitLogger_Production(t *testing.T) {
	logger, err := InitLogger("production")

	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestInitLogger_Prod(t *testing.T) {
	logger, err := InitLogger("prod")

	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestInitLogger_InvalidEnvironment(t *testing.T) {
	logger, err := InitLogger("invalid")

	require.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "unknown environment")
}

func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		Environment: "test",
		Server: ServerConfig{
			Address:      ":8080",
			ReadTimeout:  10,
			WriteTimeout: 10,
			IdleTimeout:  60,
		},
		GRPC: GRPCConfig{
			Enabled:        true,
			Address:        ":50051",
			MaxConnections: 5000,
		},
		WebSocket: WebSocketConfig{
			Enabled:      true,
			Path:         "/websocket",
			PingInterval: 15,
		},
		Kafka: KafkaConfig{
			Brokers:      []string{"kafka1:9092", "kafka2:9092"},
			Topic:        "custom-events",
			Retries:      5,
			BatchSize:    200,
			RequiredAcks: 1,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/custom-metrics",
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 500,
			BurstSize:         1000,
		},
	}

	assert.Equal(t, "test", cfg.Environment)
	assert.Equal(t, ":8080", cfg.Server.Address)
	assert.Equal(t, 2, len(cfg.Kafka.Brokers))
	assert.Equal(t, 500, cfg.RateLimit.RequestsPerSecond)
}
