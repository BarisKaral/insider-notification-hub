package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/baris/notification-hub/config"
	"github.com/baris/notification-hub/internal/app"
	"github.com/baris/notification-hub/pkg/logger"
)

// @title Notification Hub API
// @version 1.0
// @description Event-driven notification system
// @host localhost:8080
// @BasePath /api/v1
func main() {
	logger.Init("info")

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	application, err := app.NewApp(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create application")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := application.Run(); err != nil {
			logger.Fatal().Err(err).Msg("failed to run application")
		}
	}()

	logger.Info().Msg("notification-hub started")

	<-ctx.Done()

	logger.Info().Msg("shutting down...")
	if err := application.Shutdown(10 * time.Second); err != nil {
		logger.Error().Err(err).Msg("shutdown error")
	}
}
