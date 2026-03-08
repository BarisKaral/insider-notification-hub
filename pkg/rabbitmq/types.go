package rabbitmq

// RabbitMQConfig holds configuration for the RabbitMQ connection.
type RabbitMQConfig struct {
	URL string
}

// RabbitMQQueueConfig holds configuration for declaring a RabbitMQ queue.
type RabbitMQQueueConfig struct {
	Name        string
	Exchange    string
	RoutingKey  string
	DLX         string
	TTL         int // milliseconds, 0 = no TTL
	MaxPriority int // 0 = no priority
}
