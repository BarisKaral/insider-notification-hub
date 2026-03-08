package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/bariskaral/insider-notification-hub/config"
	"github.com/bariskaral/insider-notification-hub/pkg/logger"
	"github.com/bariskaral/insider-notification-hub/pkg/tracer"
)

// App is the main application struct that holds the DI container and Fiber instance.
type App struct {
	container      *Container
	fiberApp       *fiber.App
	cancelFunc     context.CancelFunc
	tracerProvider *sdktrace.TracerProvider
}

// NewApp creates a new application with all dependencies wired.
func NewApp(cfg *config.Config) (*App, error) {
	// Initialise distributed tracing.
	tracerProvider, err := tracer.InitTracer("notification-hub", cfg.Jaeger.Endpoint)
	if err != nil {
		logger.Error().Err(err).Msg("failed to init tracer, continuing without tracing")
	}

	container, err := NewContainer(cfg)
	if err != nil {
		return nil, err
	}

	application := &App{container: container, tracerProvider: tracerProvider}
	application.setupRouter()

	return application, nil
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
	return a.fiberApp.Listen(":" + a.container.Config.AppPort)
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
				if err := a.container.NotificationProcessingService.RecoverStuckNotifications(ctx); err != nil {
					logger.Error().Err(err).Msg("recovery ticker: failed to recover stuck notifications")
				}
				if err := a.container.NotificationProcessingService.PublishDueScheduled(ctx); err != nil {
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
	if err := a.fiberApp.ShutdownWithTimeout(timeout); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown fiber")
	}

	// Shutdown tracer provider to flush pending spans.
	if a.tracerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.tracerProvider.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("failed to shutdown tracer provider")
		}
	}

	// Close container (stops consumers, closes connections)
	return a.container.Close()
}
