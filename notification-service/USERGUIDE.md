# 📖 User Guide: Universal Notification Service

The **Universal Notification Service** is a production-grade, event-driven system designed to handle multi-channel notification delivery at scale. It abstracts the complexity of managing multiple providers (Email, SMS, Push, In-App) and offers advanced features like priority routing, idempotency, and distributed tracing.

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Core Concepts](#core-concepts)
3. [API Reference](#api-reference)
4. [Configuration Reference](#configuration-reference)
5. [Deployment](#deployment)
6. [Observability](#observability)

---

## System Architecture

The service follows a **Hexagonal Architecture** (Ports & Adapters), isolating the core business logic from external dependencies. The orchestrator now consumes directly from the broker with bounded worker pools, so backpressure is applied immediately to Kafka/RabbitMQ instead of buffering inside the service.

```mermaid
graph TD
    Client[Client App / Microservices] -->|HTTP / gRPC (auth)| API[API Gateway]
    Client -->|Events| Kafka[Kafka / RabbitMQ]
    
    subgraph "Notification Service"
        API --> Orchestrator
        Kafka --> Orchestrator
        
        Orchestrator -->|Direct Subscribe| Queue[Kafka / RabbitMQ]
        Orchestrator -->|Check/Set| Redis[Redis (Idempotency, Rate Limit)]
        Orchestrator -->|Persist| DB[(PostgreSQL / MongoDB)]

        Queue --> Worker[Bounded Worker Pool (50+)]
        
        Worker -->|Resolution| UserSvc[User Service]
        Worker -->|Delivery| Channels
    end
    
    subgraph "Channels (Adapters)"
        Channels -->|SMTP| EmailProvider
        Channels -->|HTTP| SMSProvider[Twilio / AWS SNS]
        Channels -->|HTTP/HTTP2| PushProvider[FCM / APNS]
        Channels -->|WS| WebSocket[In-App WebSocket]
    end
```

### Key Components

- **Orchestrator**: The central brain that coordinates the lifecycle of a notification. It handles validation, preference checks, and routing.
- **Priority Routing**: Priority fields on the event dictate broker topic / consumer group selection; the orchestrator logs the lane (“HIGH” or “DEFAULT”) for observability.
- **Worker Pool**: Bounded concurrent workers (default 50) consume directly from the broker, ensuring backpressure is applied immediately when storage or downstream providers slow down.
- **Storage Adapter**: Pluggable persistence layer supporting both PostgreSQL (relational) and MongoDB (document).
- **Security Layer**: HTTP middleware + gRPC interceptors enforce API key/JWT authentication, propagate tenant IDs, and gate recipient access consistently.

---

## Core Concepts

### 1. Unified Idempotency
To guarantee **exactly-once processing**, every request is tracked via an `Idempotency-Key`.
- **Mechanism**: Redis `SETNX` with a TTL (default 24h).
- **Behavior**: Duplicate requests within the TTL return the original response/status without re-processing.

### 2. Recipient Resolution
The service decouples user identity from contact details.
- **Strategy**: You provide a `recipient_id` (e.g., UUID).
- **Resolution**: The service queries your **User Service** (via `USER_SERVICE_URL`) to fetch email, phone number, and device tokens at runtime.
- **Override**: You can explicitly provide contact info in the payload to bypass resolution.

### 3. Smart Delivery
- **Retries**: Exponential backoff for transient failures (e.g., network timeout).
- **Dead Letter Queue (DLQ)**: Permanent failures are routed to a generic DLQ for inspection. DLQ entries now persist the original notification payload (recipient, tenant, metadata) for accurate replay.
- **Rate Limiting**: Per-user or global limits to prevent spamming.
- **Fallbacks**: Automatic SMS fallback triggers when push delivery fails and SMS is not part of the initial channel list.

### 4. Security & Multi-Tenancy

- **Authentication**:  
  - HTTP: API keys via `X-API-Key` and JWTs via `Authorization: Bearer <token>`.  
  - gRPC: Metadata-based auth with identical enforcement (either `authorization` or `x-api-key` headers).  
- **Authorization**: End-user tokens can access only their own recipient data; service/admin roles may impersonate recipients but must explicitly pass the `recipient_id`.  
- **Tenant Isolation**: Tenant IDs travel across HTTP/gRPC, are stored on every notification/delivery record, and are enforced in list/mark-read queries to prevent noisy-neighbor reads.  
- **Rate Limiting**: Configured via `RATE_LIMIT_MAX_PER_HOUR`; the service converts this to a precise per-second quota with a burst factor of 2x.

---

## API Reference

Base URL: `http://localhost:8090/v1`

### Authentication
Include the API Key in the header:
```http
X-API-Key: your-service-key
```

### Endpoints

#### 1. Create Notification
**POST** `/notifications`

Dispatches a new notification.

**Request Body:**
```json
{
  "recipient_id": "user-1234",
  "type": "ORDER_UPDATE",
  "title": "Order Shipped",
  "body": "Your package is on the way!",
  "channels": ["email", "inapp"],
  "priority": "HIGH",
  "data": {
    "order_id": "88492"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "notif-550e8400-e29b",
  "status": "queued",
  "created_at": "2023-10-27T10:00:00Z"
}
```

#### 2. Get Notification
**GET** `/notifications/{id}`

Retrieves details and status of a specific notification.

**Response (200 OK):**
```json
{
  "id": "notif-550e8400-e29b",
  "recipient_id": "user-1234",
  "status": "delivered",
  "delivery_attempts": [
    { "channel": "email", "status": "success", "timestamp": "..." }
  ]
}
```

### gRPC API

#### Stream Notifications (High Throughput)
### `StreamNotifications`
**Bi-Directional Streaming**: High-throughput ingestion.
*   **Performance**: Uses micro-batching (100 items or 50ms buffer) to achieve 50k+ RPS.
*   **Fail-Safe guarantee**: Responses are strictly correlated to requests in the batch using memory-allocated IDs.
*   **Usage**: Recommended for bulk ingestion or high-traffic event streams.

**Request (`stream CreateNotificationRequest`)**:
Same as `CreateNotification` request.

**Response (`stream CreateNotificationResponse`)**:
```protobuf
message CreateNotificationResponse {
  string id = 1;
  bool success = 2;
}
```

#### Standard RPCs
The service supports full feature parity with the HTTP API via gRPC:

| RPC Method | Equivalent HTTP | Description |
|------------|-----------------|-------------|
| `CreateNotification` | `POST /v1/notifications` | Dispatch a single notification. |
| `GetNotification` | `GET /v1/notifications/{id}` | Get details of a notification. |
| `UpdateNotification` | `PATCH /v1/notifications/{id}` | Update or mark as read. |
| `DeleteNotification` | `DELETE /v1/notifications/{id}` | Remove a notification. |
| `ListNotifications` | `GET /v1/notifications` | List notifications with pagination. |
| `BatchMarkAsRead` | `POST /v1/notifications/batch/mark-read` | Mark multiple as read. |
| `GetUnreadCount` | `GET /v1/notifications/unread-count` | Get unread badge count. |


#### 3. List Notifications
**GET** `/notifications?recipient_id=user-1234&limit=20`

Lists notifications for a user.

- **Query Params**:
    - `recipient_id` (required for non-admin)
    - `limit` (default 20)
    - `cursor` (pagination)
    - `read` (filter by read status: `true`/`false`)

#### 4. Mark as Read (Batch)
**POST** `/notifications/batch/mark-read`

Marks multiple notifications as read.

**Request Body:**
```json
{
  "recipient_id": "user-1234",
  "notification_ids": ["notif-1", "notif-2"],
  "all": false
}
```

#### 5. Get Unread Count
**GET** `/notifications/unread-count?recipient_id=user-1234`

Returns the count of unread notifications for a user.

---
## Running MAANG-Scale Load Tests

The repository ships with two complementary load suites:

1. `tests/loadtest/load_test.go` – quick smoke tests for HTTP, unary gRPC, streaming gRPC, and mixed workloads.
2. `tests/loadtest/maanf_load_test.go` – comprehensive MAANG scenarios (sustained load, spikes, stress/breaking point, streaming backpressure, and soak).

### Prerequisites
- Notification service running locally (HTTP `:8090`, gRPC `:9090` by default).  
- Real backing services (Postgres, Redis, Kafka/RabbitMQ) are recommended for accurate numbers.  
- Go 1.21+.

### Commands

```bash
# Terminal 1 – start the service
cd notification-service
SERVER_PORT=8090 GRPC_PORT=9090 STORAGE_TYPE=postgres \
QUEUE_TYPE=kafka JWT_SECRET=dev-secret \
go run ./cmd/server

# Terminal 2 – run MAANG sustained load
cd notification-service
GOCACHE=$PWD/.gocache go test ./tests/loadtest -run TestMAANG_SustainedLoad -count=1

# Optional: run spike/soak/backpressure suites
GOCACHE=$PWD/.gocache go test ./tests/loadtest -run TestMAANG_SpikeTest -count=1
```

Each run emits a structured report covering:
- Total/peak RPS
- p50/p95/p99/p99.9 latency
- Success/error rates (target ≥99.9%)
- Memory and goroutine peaks

Use these numbers to size worker pools, Kafka partitions, or to validate regression fixes before deploying.

---

## Configuration Reference

The service reads configuration from environment variables. The defaults are opinionated for local development; production deployments should override them via a secrets manager or container orchestrator.

### 1. Server & Network

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8090` | HTTP REST port. |
| `GRPC_PORT` | `9090` | gRPC port. |
| `ENVIRONMENT` | `development` | Used for logging tags. |

### 2. Storage

| Variable | Default | Notes |
|----------|---------|-------|
| `STORAGE_TYPE` | `memory` | `postgres`, `mongodb`, or `memory`. |
| `STORAGE_URI` | `""` | DSN for Postgres or Mongo. Required unless using memory. |
| `STORAGE_DATABASE` | `notifications` | MongoDB database name. |
| `STORAGE_COLLECTION` | `notifications` | MongoDB collection for notifications. |

### 3. Queue

| Variable | Default | Description |
|----------|---------|-------------|
| `QUEUE_TYPE` | `memory` | `kafka`, `rabbitmq`, or `memory`. |
| `QUEUE_BROKERS` | `localhost:9092` | Kafka brokers (`host:port`) or RabbitMQ AMQP URLs. |
| `QUEUE_TOPIC` | `notifications` | Topic/queue name for primary events. |
| `QUEUE_GROUP_ID` | `notification-service` | Consumer group ID (Kafka only). |

### 4. Redis (Idempotency, Rate Limit, Scheduler, WebSocket)

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `""` | e.g., `localhost:6379`. Required for prod features. |
| `REDIS_PASSWORD` | `""` | Optional. |
| `REDIS_DB` | `0` | Redis logical database. |

### 5. Authentication & Security

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | `""` | Required for HS256 JWT validation (HTTP + gRPC). |
| `JWT_PUBLIC_KEY` | `""` | Optional PEM for RS256 validation. |
| `API_KEYS` | `""` | Comma-separated `key:name` pairs for service-to-service auth. |

### 6. Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `true` | Toggles Redis token bucket. |
| `RATE_LIMIT_MAX_PER_HOUR` | `100` | Total allowed requests per hour; converted to per-second + burst automatically. |

### 7. Channels

Each channel has an `*_ENABLED` flag plus provider-specific settings.

- **Email**: `EMAIL_ENABLED`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM_EMAIL`, `SMTP_FROM_NAME`, `SMTP_USE_TLS`.
- **SMS (Twilio example)**: `SMS_ENABLED`, `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER`.
- **Push**: `PUSH_ENABLED`, `FCM_ENABLED`, `FCM_PROJECT_ID`, `FCM_CREDENTIALS`, `APNS_ENABLED`, `APNS_TEAM_ID`, `APNS_KEY_ID`, `APNS_BUNDLE_ID`, `APNS_KEY_FILE`.
- **In-App**: `INAPP_ENABLED`, connection limits, and optional Redis for multi-node broadcasts.

Refer to `config/config.go` for the full list and defaults.

### 8. Object Storage / Presigned URLs

Used primarily by the `storage-service`. Even if you’re consuming the notification-service only, you likely interact with MinIO/S3 via the storage microservice.

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_ENDPOINT` | `minio:9000` | Internal endpoint for server-side operations (MinIO/S3 hostname). |
| `STORAGE_SIGNER_ENDPOINT` | (defaults to `STORAGE_ENDPOINT`) | Optional override for the presign client. The signer still talks to the internal MinIO/S3 endpoint for metadata calls, but every presigned URL uses the hostname from `STORAGE_PUBLIC_URL`, so browsers only see your public/CDN host. |
| `STORAGE_PUBLIC_URL` | `http://localhost:9000` | Public URL embedded in responses so browsers/frontends can PUT/GET objects. |
| `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` | `minioadmin` | Credentials for MinIO/S3. |
| `STORAGE_BUCKET` | `connectify-uploads` | Target bucket. Created automatically if missing. |
| `STORAGE_USE_SSL` | `false` | Set to `true` for HTTPS endpoints. |

### 9. Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_ENABLED` | `true` | Exposes Prometheus metrics on `METRICS_PORT`. |
| `METRICS_PORT` | `9102` | Metrics server port. |
| `TRACING_ENABLED` | `false` | Toggles tracing exporter. |
| `JAEGER_ENDPOINT` | `""` | Target for Jaeger collector (when tracing enabled). |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |
| `LOG_FORMAT` | `json` | `json` or `console`. |

The service is configured via environment variables.

### General Server
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8090` | HTTP API port |
| `GRPC_PORT` | `50060` | gRPC API port |
| `ENVIRONMENT` | `development` | `development`, `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `USER_SERVICE_URL` | *(empty)* | URL to fetch user details (e.g. `http://user-service`) |

### Infrastructure
| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | *(required)* | Redis connection string (e.g. `localhost:6379`) |
| `STORAGE_TYPE` | `memory` | `postgres`, `mongodb`, `memory` |
| `STORAGE_URI` | *(empty)* | Database connection URI |
| `QUEUE_TYPE` | `memory` | `kafka`, `rabbitmq`, `memory` |
| `QUEUE_BROKERS` | `localhost:9092` | Comma-separated broker addresses |

### Channels
Enable or disable specific notification channels.

| Variable | Default | Logic |
|----------|---------|-------|
| `EMAIL_ENABLED` | `false` | Enable SMTP Email |
| `SMS_ENABLED` | `false` | Enable SMS (Twilio) |
| `PUSH_ENABLED` | `false` | Enable Push (FCM/APNS) |
| `INAPP_ENABLED` | `true` | Enable WebSocket In-App |

#### Email (SMTP)
Required if `EMAIL_ENABLED=true`.
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM_EMAIL`
- `SMTP_FROM_NAME`

#### Push (FCM / APNS)
Required if `PUSH_ENABLED=true`.
- `FCM_ENABLED`: `true`/`false`
- `FCM_PROJECT_ID`
- `FCM_CREDENTIALS`: JSON content of service account
- `APNS_ENABLED`: `true`/`false`
- `APNS_TEAM_ID`
- `APNS_KEY_ID`
- `APNS_BUNDLE_ID`
- `APNS_KEY_FILE`: Path to .p8 file

#### SMS (Twilio)
Required if `SMS_ENABLED=true`.
- `TWILIO_ACCOUNT_SID`
- `TWILIO_AUTH_TOKEN`
- `TWILIO_FROM_NUMBER`

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(empty)* | Secret for verifying User JWTs |
| `API_KEYS` | *(empty)* | Service-to-Service keys (`key:name,key2:name2`) |

---

## Deployment

### Docker Compose
A ready-to-use `docker-compose.yml` is provided.

```bash
# Start all services (Service + Redis + Mongo)
docker-compose up -d --build
```

### Kubernetes
Standard Kubernetes manifests are compatible. Ensure `REDIS_ADDR` and `STORAGE_URI` point to valid internal cluster services.

---

## Observability

The service exposes Prometheus metrics at `:${METRICS_PORT}/metrics` (default :9102).

**Key Metrics:**
- `notification_delivery_latency_seconds`: End-to-end delivery time.
- `queue_depth`: Number of items waiting in priority queues.
- `active_workers`: Current detailed worker count per channel.
