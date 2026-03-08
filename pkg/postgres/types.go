package postgres

import "fmt"

// PostgresConfig holds configuration for the PostgreSQL database connection.
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns the formatted connection string for the PostgreSQL database.
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
