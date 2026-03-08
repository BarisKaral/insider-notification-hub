package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort  string
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Provider ProviderConfig
	Worker   WorkerConfig
	Jaeger   JaegerConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RabbitMQConfig struct {
	URL string
}

type ProviderConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}

type WorkerConfig struct {
	Concurrency      int
	RateLimit        int
	MaxRetries       int
	RetryTTL         time.Duration
	RecoveryInterval time.Duration
}

type JaegerConfig struct {
	Endpoint string
}

func NewConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "user"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "notificationdb"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		Provider: ProviderConfig{
			URL:        getEnv("PROVIDER_URL", "https://webhook.site/your-uuid-here"),
			AuthKey:    getEnv("PROVIDER_AUTH_KEY", "secret"),
			Timeout:    getDuration("PROVIDER_TIMEOUT", 10*time.Second),
			MaxRetries: getInt("PROVIDER_MAX_RETRIES", 3),
		},
		Worker: WorkerConfig{
			Concurrency:      getInt("WORKER_CONCURRENCY", 5),
			RateLimit:        getInt("WORKER_RATE_LIMIT", 100),
			MaxRetries:       getInt("WORKER_MAX_RETRIES", 3),
			RetryTTL:         getDuration("WORKER_RETRY_TTL", 30*time.Second),
			RecoveryInterval: getDuration("WORKER_RECOVERY_INTERVAL", 30*time.Second),
		},
		Jaeger: JaegerConfig{
			Endpoint: getEnv("JAEGER_ENDPOINT", "localhost:4318"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.AppPort == "" {
		return ErrMissingAppPort
	}
	if c.Database.Host == "" || c.Database.Name == "" {
		return ErrMissingDatabaseConfig
	}
	if c.RabbitMQ.URL == "" {
		return ErrMissingRabbitMQURL
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
