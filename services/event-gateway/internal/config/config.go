package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	Environment string           `mapstructure:"environment"`
	Server      ServerConfig     `mapstructure:"server"`
	GRPC        GRPCConfig       `mapstructure:"grpc"`
	WebSocket   WebSocketConfig  `mapstructure:"websocket"`
	Kafka       KafkaConfig      `mapstructure:"kafka"`
	Metrics     MetricsConfig    `mapstructure:"metrics"`
	RateLimit   RateLimitConfig  `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Address      string `mapstructure:"address"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type GRPCConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Address         string `mapstructure:"address"`
	MaxConnections  int    `mapstructure:"max_connections"`
	MaxConcurrent   int    `mapstructure:"max_concurrent_streams"`
	ConnectionAge   int    `mapstructure:"max_connection_age"`
	KeepAliveTime   int    `mapstructure:"keepalive_time"`
	KeepAliveMinAge int    `mapstructure:"keepalive_min_age"`
}

type WebSocketConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Path         string `mapstructure:"path"`
	PingInterval int    `mapstructure:"ping_interval"`
}

type KafkaConfig struct {
	Brokers      []string `mapstructure:"brokers"`
	Topic        string   `mapstructure:"topic"`
	Retries      int      `mapstructure:"retries"`
	BatchSize    int      `mapstructure:"batch_size"`
	RequiredAcks int      `mapstructure:"required_acks"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	BurstSize         int `mapstructure:"burst_size"`
}

func Load() (*Config, error) {
	viper.SetDefault("environment", "development")

	viper.SetDefault("server.address", ":8090")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 120)

	viper.SetDefault("grpc.enabled", true)
	viper.SetDefault("grpc.address", ":9090")
	viper.SetDefault("grpc.max_connections", 10000)
	viper.SetDefault("grpc.max_concurrent_streams", 1000)
	viper.SetDefault("grpc.max_connection_age", 120)
	viper.SetDefault("grpc.keepalive_time", 10)
	viper.SetDefault("grpc.keepalive_min_age", 5)

	viper.SetDefault("websocket.enabled", false)
	viper.SetDefault("websocket.path", "/ws")
	viper.SetDefault("websocket.ping_interval", 30)

	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "events")
	viper.SetDefault("kafka.retries", 3)
	viper.SetDefault("kafka.batch_size", 100)
	viper.SetDefault("kafka.required_acks", 1)

	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.path", "/metrics")

	viper.SetDefault("rate_limit.requests_per_second", 1000)
	viper.SetDefault("rate_limit.burst_size", 2000)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/event-gateway/")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("GATEWAY")

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// InitLogger creates a logger based on the environment setting
func InitLogger(environment string) (*zap.Logger, error) {
	switch environment {
	case "production", "prod":
		return zap.NewProduction()
	case "development", "dev":
		return zap.NewDevelopment()
	default:
		return nil, fmt.Errorf("unknown environment: %s (expected: development, production, dev, or prod)", environment)
	}
}
