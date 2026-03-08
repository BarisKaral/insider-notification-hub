package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDSN(t *testing.T) {
	db := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable TimeZone=UTC"
	assert.Equal(t, expected, db.DSN())
}

func TestNewConfig_Defaults(t *testing.T) {
	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "8080", cfg.AppPort)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "user", cfg.Database.User)
	assert.Equal(t, "notificationdb", cfg.Database.Name)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
}

func TestNewConfig_WithEnvOverrides(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("WORKER_CONCURRENCY", "10")
	t.Setenv("PROVIDER_TIMEOUT", "5s")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 10, cfg.Worker.Concurrency)
	assert.Equal(t, "5s", cfg.Provider.Timeout.String())
}

func TestValidate_MissingAppPort(t *testing.T) {
	cfg := &Config{AppPort: ""}
	err := cfg.validate()
	assert.ErrorIs(t, err, ErrMissingAppPort)
}

func TestValidate_MissingDatabaseHost(t *testing.T) {
	cfg := &Config{
		AppPort:  "8080",
		Database: DatabaseConfig{Host: "", Name: ""},
	}
	err := cfg.validate()
	assert.ErrorIs(t, err, ErrMissingDatabaseConfig)
}

func TestValidate_MissingRabbitMQURL(t *testing.T) {
	cfg := &Config{
		AppPort:  "8080",
		Database: DatabaseConfig{Host: "localhost", Name: "db"},
		RabbitMQ: RabbitMQConfig{URL: ""},
	}
	err := cfg.validate()
	assert.ErrorIs(t, err, ErrMissingRabbitMQURL)
}
