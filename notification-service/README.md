# ⚡ Velox
> *The Hyperscale Universal Notification Engine*

A **production-grade, plug-and-play notification microservice** built with Go. Designed to be dropped into any project and handle millions of notifications across multiple channels.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🚀 **Multi-Channel Delivery** | In-App (WebSocket), Email (SMTP), SMS (Twilio/AWS), Push (FCM/APNS) |
| ⚡ **Priority Queues** | Fast lane for OTPs, slow lane for marketing—no head-of-line blocking |
| 🛡️ **User Governance** | Enforces user preferences (Mute, Do Not Disturb) at the infrastructure level |
| 🔄 **Exactly-Once Delivery** | Redis-backed idempotency & Transactional Outbox prevents data loss |
| 🐇 **RabbitMQ / Kafka** | Production-grade queue support with Durable Queues, Publisher Confirms, and QoS |
| 👁️ **SLO-Grade Observability** | Prometheus metrics for latency, queue depth, and delivery success rates |
| 📜 **Delivery State Machine** | Tracks every step (Pending → Sent → Delivered) per channel |
| 💀 **Dead Letter Queue** | Failed notifications are stored (DB) AND published to DLQ topics (Queue) |
| 🔐 **Dual-Mode Auth** | API Keys for services, JWT for end-users |
| 🌐 **Horizontally Scalable** | Parallel Outbox Processors & Redis Pub/Sub allow million-user scale |
| 🔌 **Pluggable Architecture** | Swap storage, queue, or user resolution easily |
| ⚙️ **Zero Hardcoded Values** | 100% configuration via environment variables |

---

## 🏗️ Production-Grade Architecture

### Key Components
1. **Parallel Outbox Processor**: Concurrent fan-out processing (up to 25k/sec per node).
2. **RabbitMQ Adapter**: Implements `amqp091` with auto-reconnect, publisher confirms, and prefetch (QoS) for backpressure.
3. **User Preference Adapter**: Governance layer that checks `EmailEnabled`, `TopicPreferences`, etc., before delivery.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         NOTIFICATION SERVICE                         │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐ │
│  │   Msg Broker    │───▶│   Orchestrator  │───▶│    Channels     │ │
│  │ (RabbitMQ/Kafka)│    │  (Parallelized) │    │  - Email/SMS    │ │
│  │   [Durable Q]   │    │  - Governance   │    │  - Push/WebSocket│ │
│  └─────────────────┘    │  - State Machine│    └─────────────────┘ │
│                         └─────────────────┘                         │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐ │
│  │  HTTP Server    │    │  User Resolver  │    │  Redis Pub/Sub  │ │
│  │  - JWT Auth     │    │  (HTTP / Mock)  │    │  - WS Broadcast │ │
│  │  - Rate Limit   │    └─────────────────┘    │  - Idempotency  │ │
│  └─────────────────┘                           └─────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### 1. Run using Docker (Recommended)

You don't need to clone the code. Just run the pre-built image:

```bash
docker run -d \
  --name notifications \
  -p 8090:8090 \
  -e STORAGE_TYPE=memory \
  -e QUEUE_TYPE=memory \
  connectify/notification-service:latest
```

Or add to your `docker-compose.yml`:

```yaml
services:
  notifications:
    image: connectify/notification-service:latest
    ports:
      - "8090:8090"
    environment:
      - STORAGE_TYPE=postgres
      - STORAGE_URI=postgres://...
      - QUEUE_TYPE=rabbitmq
      - QUEUE_BROKERS=amqp://...
```

### 2. Configure Environment

```bash
# Required for production
export REDIS_ADDR=localhost:6379
export JWT_SECRET=your-secure-secret-key
export API_KEYS=key1:service1,key2:service2

# Optional: Automatic User Resolution
export USER_SERVICE_URL=http://localhost:8080

# Storage (choose one)
export STORAGE_TYPE=postgres
export STORAGE_URI="postgres://user:pass@localhost:5432/notif?sslmode=disable"
# Or MongoDB
# export STORAGE_TYPE=mongodb
# export STORAGE_URI=mongodb://localhost:27017

# Queue (choose one - Production Recommended)
export QUEUE_TYPE=rabbitmq
export QUEUE_BROKERS=amqp://guest:guest@localhost:5672/
export QUEUE_TOPIC=notifications
# Or Kafka
# export QUEUE_TYPE=kafka
# export QUEUE_BROKERS=localhost:9092

# Observability
export METRICS_PORT=9102
```

### 2. Run with Docker

```bash
docker-compose up -d
```

### 3. Run Locally

```bash
go mod download
go run cmd/server/main.go
```

### 4. Send Your First Notification

```bash
curl -X POST http://localhost:8090/v1/notifications \
  -H "Content-Type: application/json" \
  -H "X-API-Key: key1" \
  -d '{
    "recipient_id": "user-123",
    "type": "WELCOME",
    "title": "Welcome!",
    "body": "Thanks for signing up.",
    "channels": ["inapp", "email"],
    "priority": "HIGH",
    "data": {
      "email": "user@example.com"
    }
  }'
```

---

## ⚙️ Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `REDIS_ADDR` | Redis server address | `localhost:6379` |
| `JWT_SECRET` | Secret for JWT validation | `your-secret-key` |
| `API_KEYS` | Comma-separated API keys | `key1:svc1,key2:svc2` |

### Integrations

| Variable | Default | Description |
|----------|---------|-------------|
| `USER_SERVICE_URL` | - | URL to fetch user details (HTTP GET /users/{id}) |
| `WEBHOOK_ENABLED` | `true` | Enable webhook delivery channel |

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `METRICS_PORT` | `9102` | Port for `/metrics` endpoint |
| `TRACING_ENABLED` | `false` | Enable Jaeger tracing |

### Storage & Queue

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_TYPE` | `memory` | `mongodb`, `postgres`, or `memory` |
| `QUEUE_TYPE` | `memory` | `kafka` or `memory` |
| `QUEUE_BROKERS` | `localhost:9092` | Kafka broker addresses |

See [USERGUIDE.md](./USERGUIDE.md) for complete configuration reference.

---

## 🔌 API Reference

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/notifications` | Create a notification |
| `GET` | `/v1/notifications` | List notifications for a user |
| `GET` | `/v1/notifications/{id}` | Get a specific notification |
| `PATCH` | `/v1/notifications/{id}` | Update (mark as read) |
| `DELETE` | `/v1/notifications/{id}` | Delete a notification |
| `GET` | `/v1/notifications/unread-count` | Get unread count |
| `GET` | `/health` | Health check (no auth) |
| `GET` | `/metrics` | Prometheus metrics (on METRICS_PORT) |

---

## 📦 Recipient Resolution

The service uses a pluggable `UserResolver` system:

1. **Notification Data**: Check `data.email`, `data.phone`, etc. in the request first.
2. **HTTP Resolver**: If `USER_SERVICE_URL` is set, calls your User Service (`GET /users/{id}`) to fetch details.
3. **Fallback**: Fails channel delivery if contact info is missing.

---

## 📈 Monitoring

- **Prometheus Metrics**: `http://localhost:9102/metrics`
  - `notification_delivery_latency_seconds`
  - `queue_depth`
  - `channel_delivery_failure_total`
- **Health Check**: `http://localhost:8090/health`
- **Structured Logs**: JSON format with `trace_id`

---

## 📚 Documentation

- **[USERGUIDE.md](./USERGUIDE.md)** - Detailed integration guide with examples
- **[docs/CONFIGURATION.md](./docs/CONFIGURATION.md)** - Full configuration reference

---

## 📄 License

MIT License - feel free to use in your projects!
