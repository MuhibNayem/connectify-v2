# 📖 User Guide: Universal Notification Service

This guide will walk you through integrating the Notification Service into your project—whether you're using Go, Node.js, Python, or any other language.

---

## Table of Contents

1. [Understanding the Architecture](#understanding-the-architecture)
2. [Deployment Options](#deployment-options)
3. [Required Configuration](#required-configuration)
4. [Recipient Resolution](#recipient-resolution)
5. [Delivery Lifecycle & State](#delivery-lifecycle--state)
6. [Observability & Metrics](#observability--metrics)
7. [Integration Methods](#integration-methods)
8. [Language-Specific Examples](#language-specific-examples)
9. [Configuration Reference](#configuration-reference)

---

## Understanding the Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        YOUR APPLICATION                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Auth Svc │  │ Order Svc│  │ Chat Svc │  │ Frontend │            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │             │             │             │                   │
│       └─────────────┴─────────────┴─────────────┘                   │
│                         │                                            │
│                    ┌────▼────┐                                       │
│                    │  Kafka  │ (or direct HTTP/gRPC)                 │
│                    └────┬────┘                                       │
└─────────────────────────┼───────────────────────────────────────────┘
                          │
              ┌───────────▼───────────────────────────┐
              │      NOTIFICATION SERVICE             │
              │                                       │
              │  ┌─────────────────────────────────┐ │
              │  │         Kafka Consumer          │ │
              │  │    (Commit-on-Success Only)     │ │
              │  └────────────┬────────────────────┘ │
              │               │                      │
              │  ┌────────────▼────────────────────┐ │
              │  │         Orchestrator            │ │
              │  │  • Dual Priority Queues         │ │
              │  │  • Idempotency (Redis SETNX)    │ │
              │  │  • State Machine Tracking       │ │
              │  │  • DLQ for Failed Events        │ │
              │  └────────────┬────────────────────┘ │
              │               │                      │
              │  ┌────────────▼────────────────────┐ │
              │  │          Channels               │ │
              │  │  • Email (SMTP)                 │ │
              │  │  • SMS (Twilio/AWS)             │ │
              │  │  • Push (FCM/APNS)              │ │
              │  │  • WebSocket (Redis Pub/Sub)    │ │
              │  └─────────────────────────────────┘ │
              └───────────────────────────────────────┘
```

---

## Recipient Resolution

The service needs contact details (email, phone, device tokens) to deliver messages. You can provide this in two ways:

### 1. Automatic Resolution (Recommended)

Configure the service to call your User Service automatically:

```bash
# Set this to your user service endpoint
export USER_SERVICE_URL=http://user-service:8080
```

The Notification Service will perform a `GET /users/{recipient_id}` call. It expects a JSON response like:

```json
{
  "id": "user-123",
  "email": "user@example.com",
  "phone": "+1234567890",
  "device_tokens": [
    {"token": "fcm_token_xyz", "platform": "android"}
  ]
}
```

### 2. Manual Override (Per-Notification)

You can pass contact info directly in the notification request. This **takes priority** over the resolved user data.

```json
{
  "recipient_id": "user-123",
  "channels": ["email"],
  "data": {
    "email": "override@example.com"
  }
}
```

---

## Delivery Lifecycle & State

The service tracks the precise state of every delivery attempt per channel using a persisted state machine.

### States

| State | Description |
|-------|-------------|
| `pending` | Notification received and queued |
| `sent` | Successfully sent to the provider (e.g., SMTP server accepted it) |
| `delivered` | Provider confirmed delivery (where supported) |
| `failed` | Delivery failed after max retries |
| `retrying` | Temporary failure, waiting for retry |

### Retrying

- **Transient Failures** (e.g., timeout, network): Automatically retried with exponential backoff (default: 3 retries).
- **Permanent Failures** (e.g., invalid email): fast-failed and sent to DLQ immediately.

---

## Observability & Metrics

We export SLO-grade Prometheus metrics on `:${METRICS_PORT}/metrics` (default 9102).

### Key Metrics to Monitor

| Metric Name | Type | Description |
|-------------|------|-------------|
| `notification_delivery_latency_seconds` | Histogram | End-to-end latency (p50, p95, p99). **Critical SLO metric**. |
| `channel_delivery_failure_total` | Counter | Failed deliveries by channel and error type. |
| `queue_depth` | Gauge | Current size of priority queues. High depth = scaling needed. |
| `dlq_events_total` | Counter | Number of events sent to Dead Letter Queue (failures). |
| `backpressure_events_total` | Counter | Requests rejected due to queue saturation. |

---

## Deployment Options

### Option A: Docker Compose (Recommended)

```yaml
version: '3.8'
services:
  notification-service:
    image: notification-service:latest
    ports:
      - "8090:8090"   # HTTP
      - "50060:50060" # gRPC
      - "9102:9102"   # Metrics
    environment:
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - API_KEYS=orders-svc:key1
      - USER_SERVICE_URL=http://user-service:8080
      - STORAGE_TYPE=mongodb
      - STORAGE_URI=mongodb://mongo:27017/notifications
    depends_on:
      - redis
      - mongodb
```

### Option B: Kubernetes

Deploy using standard Kubernetes manifests. Ensure you set the `REDIS_ADDR` and `JWT_SECRET` environment variables via ConfigMaps/Secrets.

---

## Integration Methods

### Method 1: REST API (Simplest)

```bash
curl -X POST http://notification-service:8090/v1/notifications \
  -H "X-API-Key: your-api-key" \
  -d '{
    "recipient_id": "user-123",
    "type": "ORDER_SHIPPED",
    "title": "Your order shipped!",
    "channels": ["inapp", "email"]
  }'
```

### Method 2: gRPC (Best Performance)

Use the generated Protobuf client to call `CreateNotification`. Ideal for inter-service communication within your cluster.

### Method 3: Kafka Events

Publish generic events to the `notifications` topic. The service consumes, deduplicates, and delivers them.

---

## Configuration Reference

### Complete Environment Variables

```bash
# Server
SERVER_PORT=8090
GRPC_PORT=50060
METRICS_PORT=9102
LOG_LEVEL=info

# Integrations
USER_SERVICE_URL=http://user-service:8080
WEBHOOK_ENABLED=true

# Auth
JWT_SECRET=your-secret
API_KEYS=key1:name1,key2:name2

# Redis (Required)
REDIS_ADDR=localhost:6379

# Storage
STORAGE_TYPE=mongodb
STORAGE_URI=mongodb://localhost:27017

# Channels
EMAIL_ENABLED=true
SMS_ENABLED=true
PUSH_ENABLED=true
INAPP_ENABLED=true
```

For channel-specific config (SMTP, Twilio, FCM), see `config/config.go`.

---

## Troubleshooting

### Notification not delivered?

1. **Check Metrics**: Is `dlq_events_total` increasing?
2. **Check Recipient**: Does `user-123` exist in your User Service? Is `USER_SERVICE_URL` correct?
3. **Check Logs**: Filter by `notification_id` or `trace_id`.

### High Latency?

1. **Check Queue Depth**: Are queues backing up?
2. **Check Redis**: Is Redis latency high? Idempotency relies on it.
3. **Check Channel Latency**: `channel_delivery_latency_seconds` will show which provider is slow.

---
