package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/baris/notification-hub/config"
	"github.com/baris/notification-hub/pkg/logger"
)

// App is the main application struct that holds the DI container and Fiber instance.
type App struct {
	container  *Container
	fiber      *fiber.App
	cancelFunc context.CancelFunc
}

// NewApp creates a new application with all dependencies wired.
func NewApp(cfg *config.Config) (*App, error) {
	container, err := NewContainer(cfg)
	if err != nil {
		return nil, err
	}

	a := &App{container: container}
	a.setupRouter()

	return a, nil
}

// Run starts consumers, recovery ticker, and the HTTP server.
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel

	// Start consumers
	if err := a.container.NotificationConsumer.Start(ctx); err != nil {
		cancel()
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start recovery ticker
	a.startRecoveryTicker(ctx)

	// Start HTTP server
	return a.fiber.Listen(":" + a.container.Config.AppPort)
}

// startRecoveryTicker periodically recovers stuck notifications and publishes due scheduled ones.
func (a *App) startRecoveryTicker(ctx context.Context) {
	ticker := time.NewTicker(a.container.Config.Worker.RecoveryInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := a.container.NotificationService.RecoverStuckNotifications(ctx); err != nil {
					logger.Error().Err(err).Msg("recovery ticker: failed to recover stuck notifications")
				}
				if err := a.container.NotificationService.PublishDueScheduled(ctx); err != nil {
					logger.Error().Err(err).Msg("recovery ticker: failed to publish due scheduled notifications")
				}
			}
		}
	}()
}

// Shutdown gracefully stops the HTTP server and closes all dependencies.
func (a *App) Shutdown(timeout time.Duration) error {
	// Cancel context to stop recovery ticker and consumers
	if a.cancelFunc != nil {
		a.cancelFunc()
	}

	// Stop HTTP server with timeout
	if err := a.fiber.ShutdownWithTimeout(timeout); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown fiber")
	}

	// Close container (stops consumers, closes connections)
	return a.container.Close()
}
