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

The service follows a **Hexagonal Architecture** (Ports & Adapters), isolating the core business logic from external dependencies.

```mermaid
graph TD
    Client[Client App / Microservices] -->|HTTP/gRPC| API[API Interface]
    Client -->|Events| Kafka[Kafka / RabbitMQ]
    
    subgraph "Notification Service"
        API --> Orchestrator
        Kafka --> Orchestrator
        
        Orchestrator -->|Priority Routing| Queue[Priority Queue]
        Orchestrator -->|Check/Set| Redis[Redis (Idempotency & Rate Limit)]
        Orchestrator -->|Persist| DB[(PostgreSQL / MongoDB)]
        
        Queue --> Worker[Worker Pool]
        
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
- **Priority Queue**: A tiered queuing system (Fast Lane vs. Slow Lane) to ensure critical alerts (e.g., OTPs) are processed before marketing messages.
- **Worker Pool**: Scalable workers that consume from the queue and execute delivery logic with retries and circuit breaking.
- **Storage Adapter**: Pluggable persistence layer supporting both PostgreSQL (relational) and MongoDB (document).

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
- **Dead Letter Queue (DLQ)**: Permanent failures are routed to a generic DLQ for inspection.
- **Rate Limiting**: Per-user or global limits to prevent spamming.

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

## Configuration Reference

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
