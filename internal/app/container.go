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
	NotificationRepository        repository.NotificationRepository
	NotificationService           service.NotificationService
	NotificationProcessingService service.NotificationProcessingService
	NotificationProducer          service.NotificationProducer
	NotificationConsumer          messaging.NotificationConsumer
	NotificationController        controller.NotificationController
	NotificationMetrics           *metrics.NotificationMetrics

	// Template domain
	NotificationTemplateRepository ntRepository.NotificationTemplateRepository
	NotificationTemplateService    ntService.NotificationTemplateService
	NotificationTemplateController ntController.NotificationTemplateController

	// Infrastructure
	HealthService health.HealthService
	HealthController  health.HealthController
	WSHub          *ws.NotificationHub
}

// NewContainer creates and wires all application dependencies.
func NewContainer(config *config.Config) (*Container, error) {
	container := &Container{Config: config}

	// 1. PostgreSQL connection
	postgresConfig := postgres.PostgresConfig{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		Name:     config.Database.Name,
		SSLMode:  config.Database.SSLMode,
	}
	database, err := postgres.NewPostgresDB(postgresConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	container.DB = database

	// 2. Run migrations
	if err := runMigrations(database); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// 3. RabbitMQ connection
	rabbitMQConnection, err := rabbitmq.NewRabbitMQConnection(rabbitmq.RabbitMQConfig{URL: config.RabbitMQ.URL})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	container.RabbitMQ = rabbitMQConnection

	// 4. Setup queues
	setupChannel, err := rabbitMQConnection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create setup channel: %w", err)
	}
	if err := messaging.SetupNotificationQueues(setupChannel); err != nil {
		return nil, fmt.Errorf("failed to setup queues: %w", err)
	}
	if err := setupChannel.Close(); err != nil {
		logger.Error().Err(err).Msg("failed to close setup channel")
	}

	// 5. Provider factory
	providerConfig := provider.ProviderConfig{
		URL:        config.Provider.URL,
		AuthKey:    config.Provider.AuthKey,
		Timeout:    config.Provider.Timeout,
		MaxRetries: config.Provider.MaxRetries,
	}
	providers := map[domain.NotificationChannel]provider.NotificationProvider{
		domain.NotificationChannelSMS: sms.NewSMSProvider(sms.SMSClientConfig{
			URL: providerConfig.URL, AuthKey: providerConfig.AuthKey,
			Timeout: providerConfig.Timeout, MaxRetries: providerConfig.MaxRetries,
		}),
		domain.NotificationChannelEmail: email.NewEmailProvider(email.EmailClientConfig{
			URL: providerConfig.URL, AuthKey: providerConfig.AuthKey,
			Timeout: providerConfig.Timeout, MaxRetries: providerConfig.MaxRetries,
		}),
		domain.NotificationChannelPush: push.NewPushProvider(push.PushClientConfig{
			URL: providerConfig.URL, AuthKey: providerConfig.AuthKey,
			Timeout: providerConfig.Timeout, MaxRetries: providerConfig.MaxRetries,
		}),
	}

	// 6. Repositories
	container.NotificationRepository = repository.NewNotificationRepository(database)
	container.NotificationTemplateRepository = ntRepository.NewNotificationTemplateRepository(database)

	// 7. Producer (needs AMQP channel)
	producerChannel, err := rabbitMQConnection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create producer channel: %w", err)
	}
	container.NotificationProducer = messaging.NewNotificationProducer(producerChannel)

	// 8. Services
	container.NotificationTemplateService = ntService.NewNotificationTemplateService(container.NotificationTemplateRepository)
	container.NotificationService = service.NewNotificationService(container.NotificationRepository, container.NotificationTemplateService)
	container.NotificationProcessingService = service.NewNotificationProcessingService(
		container.NotificationService,
		container.NotificationProducer,
		providers,
		container.NotificationRepository,
	)

	// 9. WebSocket Hub
	container.WSHub = ws.NewNotificationHub()

	// 10. Metrics
	container.NotificationMetrics = metrics.NewNotificationMetrics()

	// 11. Consumer (needs AMQP channel)
	consumerChannel, err := rabbitMQConnection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer channel: %w", err)
	}
	consumerConfig := messaging.NotificationConsumerConfig{
		Concurrency: config.Worker.Concurrency,
		RateLimit:   config.Worker.RateLimit,
		MaxRetries:  config.Worker.MaxRetries,
		RetryTTL:    config.Worker.RetryTTL,
	}
	container.NotificationConsumer = messaging.NewNotificationConsumer(
		container.NotificationProcessingService,
		consumerChannel,
		container.WSHub,
		container.NotificationMetrics,
		consumerConfig,
	)

	// 12. Controllers
	container.NotificationController = controller.NewNotificationController(container.NotificationService, container.NotificationProcessingService)
	container.NotificationTemplateController = ntController.NewNotificationTemplateController(container.NotificationTemplateService)

	// 13. Health check
	container.HealthService = health.NewHealthService(database, rabbitMQConnection)
	container.HealthController = health.NewHealthController(container.HealthService)

	logger.Info().Msg("all dependencies initialized")

	return container, nil
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
		sqlDatabase, err := c.DB.DB()
		if err == nil {
			if err := sqlDatabase.Close(); err != nil {
				logger.Error().Err(err).Msg("failed to close database connection")
			}
		}
	}

	logger.Info().Msg("all dependencies closed")
	return nil
}

// runMigrations runs database migrations using golang-migrate.
func runMigrations(db *gorm.DB) error {
	sqlDatabase, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	driver, err := migratePostgres.WithInstance(sqlDatabase, &migratePostgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	migrateInstance, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := migrateInstance.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info().Msg("database migrations applied")
	return nil
}
