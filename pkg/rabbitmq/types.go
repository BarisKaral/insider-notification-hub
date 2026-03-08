package rabbitmq

type QueueConfig struct {
	Name        string
	Exchange    string
	RoutingKey  string
	DLX         string
	TTL         int // milliseconds, 0 = no TTL
	MaxPriority int // 0 = no priority
}
