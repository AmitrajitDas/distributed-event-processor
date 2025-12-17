package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	Environment string            `mapstructure:"environment"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Processing  ProcessingConfig  `mapstructure:"processing"`
	Metrics     MetricsConfig     `mapstructure:"metrics"`
	Health      HealthConfig      `mapstructure:"health"`
}

type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	Topic         string   `mapstructure:"topic"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	AutoCommit    bool     `mapstructure:"auto_commit"`
	StartOffset   string   `mapstructure:"start_offset"` // earliest, latest
}

type RedisConfig struct {
	Address    string `mapstructure:"address"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	PoolSize   int    `mapstructure:"pool_size"`
	MaxRetries int    `mapstructure:"max_retries"`
}

type ProcessingConfig struct {
	Workers        int    `mapstructure:"workers"`
	WindowType     string `mapstructure:"window_type"`      // tumbling, sliding, session
	WindowSize     int    `mapstructure:"window_size"`      // seconds
	WindowSlide    int    `mapstructure:"window_slide"`     // seconds (for sliding windows)
	SessionGap     int    `mapstructure:"session_gap"`      // seconds (for session windows)
	WatermarkDelay int    `mapstructure:"watermark_delay"`  // seconds
	MaxOutOfOrder  int    `mapstructure:"max_out_of_order"` // seconds
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

type HealthConfig struct {
	Port int    `mapstructure:"port"`
	Path string `mapstructure:"path"`
}

func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("environment", "development")

	// Kafka defaults
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "events")
	viper.SetDefault("kafka.consumer_group", "stream-processor")
	viper.SetDefault("kafka.auto_commit", false)
	viper.SetDefault("kafka.start_offset", "latest")

	// Redis defaults
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.max_retries", 3)

	// Processing defaults
	viper.SetDefault("processing.workers", 4)
	viper.SetDefault("processing.window_type", "tumbling")
	viper.SetDefault("processing.window_size", 300)      // 5 minutes
	viper.SetDefault("processing.window_slide", 60)      // 1 minute
	viper.SetDefault("processing.session_gap", 600)      // 10 minutes
	viper.SetDefault("processing.watermark_delay", 10)   // 10 seconds
	viper.SetDefault("processing.max_out_of_order", 60) // 1 minute

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9091)
	viper.SetDefault("metrics.path", "/metrics")

	// Health defaults
	viper.SetDefault("health.port", 8091)
	viper.SetDefault("health.path", "/health")

	// Configuration file settings
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/stream-processor/")

	// Environment variable settings
	viper.AutomaticEnv()
	viper.SetEnvPrefix("PROCESSOR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (if exists)
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Unmarshal config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validate performs basic validation on the configuration
func validate(cfg *Config) error {
	// Validate Kafka config
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("kafka topic cannot be empty")
	}
	if cfg.Kafka.ConsumerGroup == "" {
		return fmt.Errorf("kafka consumer group cannot be empty")
	}
	if cfg.Kafka.StartOffset != "earliest" && cfg.Kafka.StartOffset != "latest" {
		return fmt.Errorf("kafka start_offset must be 'earliest' or 'latest'")
	}

	// Validate Redis config
	if cfg.Redis.Address == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	// Validate Processing config
	if cfg.Processing.Workers < 1 {
		return fmt.Errorf("processing workers must be at least 1")
	}
	if cfg.Processing.WindowType != "tumbling" && cfg.Processing.WindowType != "sliding" && cfg.Processing.WindowType != "session" {
		return fmt.Errorf("processing window_type must be 'tumbling', 'sliding', or 'session'")
	}
	if cfg.Processing.WindowSize < 1 {
		return fmt.Errorf("processing window_size must be at least 1 second")
	}
	if cfg.Processing.WindowType == "sliding" && cfg.Processing.WindowSlide < 1 {
		return fmt.Errorf("processing window_slide must be at least 1 second for sliding windows")
	}
	if cfg.Processing.WindowType == "session" && cfg.Processing.SessionGap < 1 {
		return fmt.Errorf("processing session_gap must be at least 1 second for session windows")
	}

	return nil
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
