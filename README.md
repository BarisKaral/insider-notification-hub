# Notification Hub

An event-driven notification system built with Go that processes and delivers messages through **SMS**, **Email**, and **Push** channels. Features async processing via RabbitMQ, automatic retry with dead-letter queues, rate limiting, priority queues, real-time WebSocket status updates, a template system, and full observability.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| HTTP Framework | Fiber v2 |
| ORM | GORM + PostgreSQL |
| Message Broker | RabbitMQ (amqp091-go) |
| Logging | zerolog |
| Tracing | OpenTelemetry + Jaeger |
| Metrics | Prometheus + Grafana |
| API Docs | Swagger (swaggo/swag) |
| Migrations | golang-migrate |
| Rate Limiting | golang.org/x/time/rate |

## Architecture

```
Client
  │
  POST /api/v1/notifications
  │
  ▼
Controller ──► Service ──► Repository (PostgreSQL)
                  │
                  ▼
              Producer ──► RabbitMQ Exchange
                              │
                 ┌─────────────┼─────────────┐
                 ▼             ▼             ▼
           queue.sms     queue.email    queue.push
                 │             │             │
                 ▼             ▼             ▼
           Consumer       Consumer       Consumer
           (rate limited) (rate limited) (rate limited)
                 │             │             │
                 ▼             ▼             ▼
           Provider        Provider       Provider
           (webhook)       (webhook)      (webhook)
                 │
            on failure
                 ▼
           DLQ ──► Retry Queue (TTL 30s) ──► Main Queue
```

### Queue Topology

3 exchanges and 9 queues:

| Exchange | Type | Purpose |
|----------|------|---------|
| `notification.exchange` | direct | Main routing to channel queues |
| `notification.dlx` | direct | Dead letter routing |
| `notification.retry.exchange` | direct | Retry routing with TTL |

| Queue | Purpose |
|-------|---------|
| `notification.queue.{sms,email,push}` | Main processing (priority-enabled) |
| `notification.dlq.{sms,email,push}` | Dead letter queues |
| `notification.retry.{sms,email,push}` | Retry with 30s TTL |

### Retry Logic

1. Consumer fails to send → NACK → message routed to DLQ via dead-letter exchange
2. DLQ consumer checks `x-retry-count` header
3. If retries remaining (default max: 3) → publish to retry queue with 30s TTL
4. TTL expires → message routes back to main queue for reprocessing
5. Max retries exceeded → marked as permanently failed

### Recovery Ticker

Runs every 30 seconds to handle:
- **Stuck pending**: Notifications where queue publish failed
- **Due scheduled**: Scheduled notifications whose delivery time has arrived

## Quick Start

### Prerequisites

- Docker & Docker Compose

### Run

```bash
# Build all images
make install

# Start all services
make up

# Stop all services
make down
```

No external dependencies — a built-in fake webhook server handles provider callbacks. Database migrations run automatically on first start.

This starts:
- **App** at `http://localhost:8080`
- **Fake Webhook** at `http://localhost:8081` (provider mock)
- **PostgreSQL** at `localhost:5432`
- **RabbitMQ** at `localhost:5672` (Management UI: `http://localhost:15672`)
- **Prometheus** at `http://localhost:9090`
- **Grafana** at `http://localhost:3000` (admin/admin)
- **Jaeger** at `http://localhost:16686`
- **pgAdmin** at `http://localhost:5050` (desktop mode, no login required)

## API Examples

### Create a Notification

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-key-123" \
  -d '{
    "recipient": "+905551234567",
    "channel": "sms",
    "content": "Hello, your order has been shipped!",
    "priority": "high"
  }'
```

### Create with Template

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "user@example.com",
    "channel": "email",
    "templateId": "<template-uuid>",
    "variables": {"name": "Baris", "orderId": "12345"}
  }'
```

### Create Scheduled Notification

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+905551234567",
    "channel": "sms",
    "content": "Scheduled reminder",
    "scheduledAt": "2026-03-09T10:00:00Z"
  }'
```

### Create Batch

```bash
curl -X POST http://localhost:8080/api/v1/notifications/batch \
  -H "Content-Type: application/json" \
  -d '{
    "notifications": [
      {"recipient": "+905551234567", "channel": "sms", "content": "Message 1", "priority": "high"},
      {"recipient": "user@example.com", "channel": "email", "content": "Message 2"}
    ]
  }'
```

### Get Notification

```bash
curl http://localhost:8080/api/v1/notifications/<notification-id>
```

### List Notifications (with filters)

```bash
curl "http://localhost:8080/api/v1/notifications?status=sent&channel=sms&limit=20&offset=0"
```

### Get Batch

```bash
curl http://localhost:8080/api/v1/notifications/batch/<batch-id>
```

### Cancel Notification

```bash
curl -X PATCH http://localhost:8080/api/v1/notifications/<notification-id>/cancel
```

### Template CRUD

```bash
# Create template
curl -X POST http://localhost:8080/api/v1/notification-templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "order_shipped",
    "channel": "sms",
    "content": "Hello {{name}}, your order {{orderId}} has been shipped."
  }'

# Get template
curl http://localhost:8080/api/v1/notification-templates/<template-id>

# List templates
curl "http://localhost:8080/api/v1/notification-templates?limit=20&offset=0"

# Update template
curl -X PUT http://localhost:8080/api/v1/notification-templates/<template-id> \
  -H "Content-Type: application/json" \
  -d '{
    "name": "order_shipped_v2",
    "channel": "sms",
    "content": "Hi {{name}}, order {{orderId}} is on the way!"
  }'

# Delete template
curl -X DELETE http://localhost:8080/api/v1/notification-templates/<template-id>
```

### Health Check

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "checks": {
    "database": "up",
    "rabbitmq": "up"
  }
}
```

## WebSocket

Subscribe to real-time notification status updates:

```bash
# Single notification
wscat -c ws://localhost:8080/ws/notifications/<notification-id>

# All notifications in a batch
wscat -c ws://localhost:8080/ws/notifications/batch/<batch-id>
```

Server sends status updates:

```json
{
  "notificationId": "uuid",
  "status": "sent",
  "timestamp": "2026-03-08T12:01:00Z"
}
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `8080` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `user` | PostgreSQL user |
| `DB_PASSWORD` | `password` | PostgreSQL password |
| `DB_NAME` | `notificationdb` | PostgreSQL database name |
| `DB_SSL_MODE` | `disable` | PostgreSQL SSL mode |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |
| `PROVIDER_URL` | `http://fakewebhook:8081` | Webhook provider URL |
| `PROVIDER_AUTH_KEY` | - | Provider authentication key |
| `PROVIDER_TIMEOUT` | `10s` | Provider HTTP timeout |
| `PROVIDER_MAX_RETRIES` | `3` | Max delivery retries |
| `WORKER_CONCURRENCY` | `5` | Consumer goroutines per channel |
| `WORKER_RATE_LIMIT` | `100` | Messages per second per channel |
| `WORKER_MAX_RETRIES` | `3` | Max retry attempts |
| `WORKER_RETRY_TTL` | `30s` | Retry queue TTL |
| `WORKER_RECOVERY_INTERVAL` | `30s` | Recovery ticker interval |
| `JAEGER_ENDPOINT` | `http://localhost:14268/api/traces` | Jaeger collector endpoint |

## Testing

```bash
# Unit tests (Docker)
make test

# E2E tests (requires running services)
make test-e2e

# Lint
make lint
```

## Monitoring

| Service | URL | Purpose |
|---------|-----|---------|
| Grafana | http://localhost:3000 | Dashboards (admin/admin) |
| Prometheus | http://localhost:9090 | Metrics queries |
| Jaeger | http://localhost:16686 | Distributed tracing |
| RabbitMQ | http://localhost:15672 | Queue management (guest/guest) |
| pgAdmin | http://localhost:5050 | PostgreSQL client (no login) |
| Swagger | http://localhost:8080/swagger/ | API documentation |

### Prometheus Metrics

- `notifications_total` — Counter of notifications processed by channel and status
- `notifications_processing_duration_seconds` — Histogram of processing duration by channel
- `queue_depth` — Gauge of current notification queue depth by channel

## Project Structure

```
notification-hub/
├── cmd/api/main.go                          # Entry point (HTTP + Workers)
├── cmd/fakewebhook/main.go                  # Fake provider webhook server
├── config/
│   ├── config.go                            # Env parsing & validation
│   └── errors.go
├── internal/
│   ├── app/
│   │   ├── app.go                           # Lifecycle (Run / Shutdown)
│   │   ├── container.go                     # DI wiring
│   │   └── router.go                        # Routes + error handler
│   ├── notification/
│   │   ├── controller/controller.go         # HTTP endpoints
│   │   ├── domain/                          # Model, DTOs, enums, errors
│   │   ├── messaging/                       # Producer, consumer, topology
│   │   ├── metrics/                         # Prometheus instrumentation
│   │   ├── provider/{sms,email,push}/       # Channel-specific providers
│   │   ├── repository/repository.go         # PostgreSQL (GORM)
│   │   ├── service/                         # Business logic
│   │   └── ws/hub.go                        # WebSocket hub
│   └── notificationtemplate/
│       ├── controller/controller.go
│       ├── domain/
│       ├── repository/repository.go
│       └── service/service.go
├── pkg/
│   ├── errs/                                # Structured app errors
│   ├── health/                              # Health check endpoint
│   ├── httpclient/                          # HTTP client with retry
│   ├── logger/                              # zerolog wrapper
│   ├── middleware/                           # Recover, CORS, RequestID
│   ├── postgres/                            # DB connection
│   ├── rabbitmq/                            # RabbitMQ connection
│   ├── response/                            # JSON response envelope
│   └── tracer/                              # OpenTelemetry setup
├── migrations/                              # SQL migration files
├── seed/seeder.go                           # Seed data
├── test/e2e/                                # End-to-end tests
├── deploy/
│   ├── prometheus/prometheus.yml
│   ├── grafana/                             # Datasources & dashboards
│   └── pgadmin/servers.json                 # Auto-configured server
├── docs/                                    # Swagger (auto-generated)
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── .github/workflows/ci.yml                 # CI pipeline
```

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Monolith** | Single binary simplifies deployment; domain separation allows future extraction |
| **Per-channel queues** | Independent scaling and rate limiting per notification channel |
| **DLQ + Retry Queue (TTL)** | Clean separation of concerns; retry delay without blocking consumers |
| **Consumer-side rate limiting** | Token bucket (100 msg/sec/channel) prevents provider overload |
| **Idempotency key** | Optional client-provided key with unique DB constraint; server UUID if absent |
| **SELECT FOR UPDATE** | Prevents duplicate delivery when same message consumed twice |
| **Recovery ticker** | Catches failed queue publishes and due scheduled notifications |
| **WebSocket per notification/batch** | Real-time status tracking without polling |
| **Template system** | `{{variable}}` replacement; decouples message content from notification logic |
| **Boundary-only tracing** | OpenTelemetry spans at HTTP/DB/RabbitMQ/Provider boundaries; minimal overhead |
