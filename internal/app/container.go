package app

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"

	"github.com/baris/notification-hub/config"
	"github.com/baris/notification-hub/internal/notification/controller"
	"github.com/baris/notification-hub/internal/notification/messaging"
	"github.com/baris/notification-hub/internal/notification/metrics"
	"github.com/baris/notification-hub/internal/notification/repository"
	"github.com/baris/notification-hub/internal/notification/service"
	"github.com/baris/notification-hub/internal/notification/ws"
	ntController "github.com/baris/notification-hub/internal/notificationtemplate/controller"
	ntRepository "github.com/baris/notification-hub/internal/notificationtemplate/repository"
	ntService "github.com/baris/notification-hub/internal/notificationtemplate/service"
	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/provider"
	"github.com/baris/notification-hub/internal/notification/provider/email"
	"github.com/baris/notification-hub/internal/notification/provider/push"
	"github.com/baris/notification-hub/internal/notification/provider/sms"
	"github.com/baris/notification-hub/pkg/health"
	"github.com/baris/notification-hub/pkg/logger"
	"github.com/baris/notification-hub/pkg/postgres"
	"github.com/baris/notification-hub/pkg/rabbitmq"
)

// Container holds all application dependencies.
type Container struct {
	Config   *config.Config
	DB       *gorm.DB
	RabbitMQ rabbitmq.RabbitMQConnection

	// Notification domain
	NotificationRepo              repository.NotificationRepository
	NotificationService           service.NotificationService
	NotificationProcessingService service.NotificationProcessingService
	NotificationProducer          service.NotificationProducer
	NotificationConsumer          messaging.NotificationConsumer
	NotificationController        controller.NotificationController
	NotificationMetrics           *metrics.NotificationMetrics

	// Template domain
	TemplateRepo       ntRepository.NotificationTemplateRepository
	TemplateService    ntService.NotificationTemplateService
	TemplateController ntController.NotificationTemplateController

	// Infrastructure
	HealthService health.HealthService
	HealthController  health.HealthController
	WSHub          *ws.NotificationHub
}

// NewContainer creates and wires all application dependencies.
func NewContainer(cfg *config.Config) (*Container, error) {
	c := &Container{Config: cfg}

	// 1. PostgreSQL connection
	pgCfg := postgres.PostgresConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}
	db, err := postgres.NewPostgresDB(pgCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	c.DB = db

	// 2. Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// 3. RabbitMQ connection
	rmqConn, err := rabbitmq.NewRabbitMQConnection(rabbitmq.RabbitMQConfig{URL: cfg.RabbitMQ.URL})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	c.RabbitMQ = rmqConn

	// 4. Setup queues
	setupCh, err := rmqConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create setup channel: %w", err)
	}
	if err := messaging.SetupNotificationQueues(setupCh); err != nil {
		return nil, fmt.Errorf("failed to setup queues: %w", err)
	}
	if err := setupCh.Close(); err != nil {
		logger.Error().Err(err).Msg("failed to close setup channel")
	}

	// 5. Provider factory
	providerCfg := provider.ProviderConfig{
		URL:        cfg.Provider.URL,
		AuthKey:    cfg.Provider.AuthKey,
		Timeout:    cfg.Provider.Timeout,
		MaxRetries: cfg.Provider.MaxRetries,
	}
	providers := map[domain.NotificationChannel]provider.NotificationProvider{
		domain.NotificationChannelSMS: sms.NewSMSProvider(sms.SMSClientConfig{
			URL: providerCfg.URL, AuthKey: providerCfg.AuthKey,
			Timeout: providerCfg.Timeout, MaxRetries: providerCfg.MaxRetries,
		}),
		domain.NotificationChannelEmail: email.NewEmailProvider(email.EmailClientConfig{
			URL: providerCfg.URL, AuthKey: providerCfg.AuthKey,
			Timeout: providerCfg.Timeout, MaxRetries: providerCfg.MaxRetries,
		}),
		domain.NotificationChannelPush: push.NewPushProvider(push.PushClientConfig{
			URL: providerCfg.URL, AuthKey: providerCfg.AuthKey,
			Timeout: providerCfg.Timeout, MaxRetries: providerCfg.MaxRetries,
		}),
	}

	// 6. Repositories
	c.NotificationRepo = repository.NewNotificationRepository(db)
	c.TemplateRepo = ntRepository.NewNotificationTemplateRepository(db)

	// 7. Producer (needs AMQP channel)
	producerCh, err := rmqConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create producer channel: %w", err)
	}
	c.NotificationProducer = messaging.NewNotificationProducer(producerCh)

	// 8. Services
	c.TemplateService = ntService.NewNotificationTemplateService(c.TemplateRepo)
	c.NotificationService = service.NewNotificationService(c.NotificationRepo, c.TemplateService, c.NotificationProducer)
	c.NotificationProcessingService = service.NewNotificationProcessingService(
		c.NotificationService,
		c.NotificationProducer,
		providers,
	)

	// 9. WebSocket Hub
	c.WSHub = ws.NewNotificationHub()

	// 10. Metrics
	c.NotificationMetrics = metrics.NewNotificationMetrics()

	// 11. Consumer (needs AMQP channel)
	consumerCh, err := rmqConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer channel: %w", err)
	}
	consumerCfg := messaging.NotificationConsumerConfig{
		Concurrency: cfg.Worker.Concurrency,
		RateLimit:   cfg.Worker.RateLimit,
		MaxRetries:  cfg.Worker.MaxRetries,
		RetryTTL:    cfg.Worker.RetryTTL,
	}
	c.NotificationConsumer = messaging.NewNotificationConsumer(
		c.NotificationProcessingService,
		consumerCh,
		c.WSHub,
		c.NotificationMetrics,
		consumerCfg,
	)

	// 12. Controllers
	c.NotificationController = controller.NewNotificationController(c.NotificationService, c.NotificationProcessingService)
	c.TemplateController = ntController.NewNotificationTemplateController(c.TemplateService)

	// 13. Health check
	c.HealthService = health.NewHealthService(db, rmqConn)
	c.HealthController = health.NewHealthController(c.HealthService)

	logger.Info().Msg("all dependencies initialized")

	return c, nil
}

// Close shuts down all dependencies in reverse order.
func (c *Container) Close() error {
	// Stop consumer
	if c.NotificationConsumer != nil {
		if err := c.NotificationConsumer.Stop(); err != nil {
			logger.Error().Err(err).Msg("failed to stop consumer")
		}
	}

	// Close RabbitMQ
	if c.RabbitMQ != nil {
		if err := c.RabbitMQ.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close rabbitmq connection")
		}
	}

	// Close database
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Error().Err(err).Msg("failed to close database connection")
			}
		}
	}

	logger.Info().Msg("all dependencies closed")
	return nil
}

// runMigrations runs database migrations using golang-migrate.
func runMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	driver, err := migratePostgres.WithInstance(sqlDB, &migratePostgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info().Msg("database migrations applied")
	return nil
}
