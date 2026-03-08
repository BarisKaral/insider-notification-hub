package postgres

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDB(config PostgresConfig) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDatabase, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDatabase.SetMaxIdleConns(10)
	sqlDatabase.SetMaxOpenConns(100)
	sqlDatabase.SetConnMaxLifetime(time.Hour)

	return database, nil
}
