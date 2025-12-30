# Universal Notification Service

A robust, event-driven notification orchestration engine designed for high-scale microservices architectures.

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/MuhibNayem/connectify-v2/notification-service)](https://goreportcard.com/report/github.com/MuhibNayem/connectify-v2/notification-service)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## 🚀 Overview

This service standardizes how notifications (Email, SMS, Push, In-App) are dispatched across your entire platform. Instead of every microservice implementing its own email client or Twilio integration, they simply push an event to this service, which handles:

- **Orchestration**: Priority routing, templating, and user preference checks.
- **Reliability**: Two-phase idempotency, exact-once worker locks, DLQ persistence, and smart channel fallback.
- **Scalability**: Direct Kafka/RabbitMQ backpressure with bounded concurrency pools (50+ workers per pod) validated by MAANG-scale load tests.

## ✨ Key Features

- **Multi-Channel Support**: Email (SMTP), SMS (Twilio), Push (FCM/APNS), In-App (WebSocket).
- **Priority Awareness**: Broker-level fast lanes (topic/priority metadata) with deterministic worker QoS.
- **Idempotency**: Guarantees exactly-once processing using Redis locks + confirmation writes.
- **Pluggable Architecture**: Easily swap storage (Postgres/Mongo) and queues (Kafka/RabbitMQ).
- **Observability & Governance**: Built-in Prometheus metrics, structured tracing, tenant-aware auth (API keys + JWTs), and rate limiting.

## 📖 Documentation

For detailed integration instructions, API references, and configuration options, please read the **[User Guide](USERGUIDE.md)**.

## ⚡ Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 14+ and/or MongoDB 6+ (optional, the service defaults to in-memory)
- Kafka/RabbitMQ (optional; defaults to in-memory queue for dev)
- Redis 7+ (required for idempotency, rate limiting, scheduler, and WebSocket fan-out)

### Run Locally

1. **Clone the repository:**
   ```bash
   git clone https://github.com/MuhibNayem/connectify-v2.git
   cd connectify-v2/notification-service
   ```

2. **Start dependencies and service:**
   ```bash
   docker-compose up -d
   ```

3. **Run migrations (optional for Postgres/Mongo):**
   ```bash
   go run ./cmd/migrate up
   ```

4. **Send a test notification (HTTP):**
   ```bash
   curl -X POST http://localhost:8090/v1/notifications \
     -H "X-API-Key: dev-key" \
     -d '{
       "recipient_id": "test-user",
       "type": "WELCOME",
       "channels": ["inapp"],
       "title": "Hello World",
       "body": "This is a test notification."
     }'
   ```

5. **Send a test notification (gRPC):**
   ```bash
   grpcurl -plaintext -H "authorization: Bearer dev-token" \
     -d '{"recipient_id":"test-user","type":"WELCOME","title":"Hi","body":"Hello from gRPC"}' \
     localhost:9090 notification.v1.NotificationService/CreateNotification
   ```

### Run the MAANG Load Suite

The repository ships with a production-grade load suite that stresses both HTTP and gRPC endpoints.

```bash
# Terminal 1: start the service (with Postgres/Redis/Kafka for best fidelity)
SERVER_PORT=8090 GRPC_PORT=9090 go run ./cmd/server

# Terminal 2: run the suite (requires Go 1.21+)
cd notification-service
GOCACHE=$PWD/.gocache go test ./tests/loadtest -run TestMAANG_SustainedLoad -count=1
```

The test prints a full MAANG report (RPS, p50/p95/p99 latency, success rate, memory, goroutines). Additional scenarios such as spike, stress, streaming backpressure, and 60s soak are available by running the other `TestMAANG_*` cases.

## 🛠️ Tech Stack

- **Language**: Go
- **Messaging**: Kafka, RabbitMQ
- **Storage**: PostgreSQL, MongoDB
- **Cache**: Redis
- **Observability**: Prometheus, Jaeger

## 🛡️ Security & Multi-Tenancy

- **HTTP**: Supports API keys and JWTs with tenant claims. End-user tokens are enforced per-recipient while service accounts may act on behalf of any user.
- **gRPC**: Uses metadata-based auth via interceptors (`authorization` header or `x-api-key`) so both unary and streaming calls are protected identically.
- **Rate Limiting**: Redis-backed token bucket with precise per-second quotas derived from hourly caps.
- **Tenant Propagation**: Tenant IDs are persisted on every notification/state record to guarantee isolation in Postgres/Mongo queries.

See `USERGUIDE.md` for full configuration knobs.

## ⚙️ Environment Configuration

All configuration is driven by environment variables (or a `.env` file if you prefer). The most commonly used settings:

| Category | Variables | Description |
|----------|-----------|-------------|
| Server | `SERVER_PORT` (default `8090`), `GRPC_PORT` (default `9090`), `ENVIRONMENT` | HTTP/gRPC listening ports and environment label. |
| Storage | `STORAGE_TYPE` (`postgres`, `mongodb`, `memory`), `STORAGE_URI`, `STORAGE_DATABASE`, `STORAGE_COLLECTION` | Select and configure the storage adapter. Use the Postgres DSN or Mongo URI when not using memory. |
| Queue | `QUEUE_TYPE` (`kafka`, `rabbitmq`, `memory`), `QUEUE_BROKERS` (comma-separated), `QUEUE_TOPIC`, `QUEUE_GROUP_ID` | Configure which queue adapter to use. Brokers are `host:port` for Kafka or AMQP URLs for RabbitMQ. |
| Redis | `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` | Shared Redis connection used for idempotency, rate limiting, scheduler, and WebSocket pub/sub. |
| Channels | `EMAIL_ENABLED`, `SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMS_ENABLED`, `TWILIO_ACCOUNT_SID`, etc. | Toggle each channel and provide provider credentials. See `config/config.go` for the full list. |
| Auth | `JWT_SECRET`, `JWT_PUBLIC_KEY`, `API_KEYS` (`key1:name1,key2:name2`) | Required for production deployments. HTTP and gRPC share the same auth config. |
| Rate Limit | `RATE_LIMIT_ENABLED`, `RATE_LIMIT_MAX_PER_HOUR` | Controls Redis-backed token bucket configuration. |
| Observability | `METRICS_ENABLED`, `METRICS_PORT`, `TRACING_ENABLED`, `JAEGER_ENDPOINT` | Toggle metrics/tracing export. |

Example `.env` (development):
```env
SERVER_PORT=8090
GRPC_PORT=9090
STORAGE_TYPE=postgres
STORAGE_URI=postgres://notif:notif@localhost:5432/notifications?sslmode=disable
QUEUE_TYPE=kafka
QUEUE_BROKERS=localhost:9092
QUEUE_TOPIC=notifications
QUEUE_GROUP_ID=notification-service
REDIS_ADDR=localhost:6379
JWT_SECRET=dev-secret
API_KEYS=dev-key:internal
RATE_LIMIT_MAX_PER_HOUR=3600
EMAIL_ENABLED=true
SMTP_HOST=smtp.sendgrid.net
SMTP_USERNAME=apikey
SMTP_PASSWORD=sg-xxx
```

> Tip: Keep secrets (JWT, SMTP, Twilio, etc.) outside VCS by using a `.env.local` or your deployment platform’s secret manager.
> A starter `.env.example` lives in the repo for convenience.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
