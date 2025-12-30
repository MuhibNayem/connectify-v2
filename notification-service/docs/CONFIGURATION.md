# ⚙️ Configuration Reference

The Universal Notification Service is designed to be **12-factor app compliant**, meaning all configuration is handled via environment variables.

> [!NOTE]
> Defaults are optimized for development (`memory` storage/queue). For production, you **must** override storage and queue settings.

---

## 1. Core Server Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | `development`, `staging`, `production`. Controls logging verbosity and safety checks. |
| `SERVER_PORT` | `8090` | Port for the HTTP API. |
| `GRPC_PORT` | `50060` | Port for the gRPC API. |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `json` | `json` for production (Splunk/ELK), `console` for local dev. |

---

## 2. Infrastructure (Required for Prod)

### Storage (Database)
Controls where notification status and history is stored.

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_TYPE` | `memory` | `postgres`, `mongodb`, `memory`. |
| `STORAGE_URI` | *(empty)* | Connection string (e.g., `postgres://user:pass@host:5432/db` or `mongodb://mongo:27017`). |
| `STORAGE_DATABASE` | `notifications` | Database name (MongoDB only). |
| `STORAGE_COLLECTION` | `notifications` | Collection name (MongoDB only). |

### Queue (Async Processing)
Controls how events are buffered and processed.

| Variable | Default | Description |
|----------|---------|-------------|
| `QUEUE_TYPE` | `memory` | `kafka`, `rabbitmq`, `memory`. |
| `QUEUE_BROKERS` | `localhost:9092` | Comma-separated list of brokers (Kafka/RabbitMQ). |
| `QUEUE_TOPIC` | `notifications` | Main topic/exchange name. |
| `QUEUE_GROUP_ID` | `notification-service` | Consumer group ID (for scaling workers). |

### Redis (Cache & Idempotency)
Required for idempotency, rate limiting, and in-app WebSocket channels.

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | *(empty)* | Host:Port (e.g., `localhost:6379`). |
| `REDIS_PASSWORD` | *(empty)* | Optional password. |
| `REDIS_DB` | `0` | Redis database index. |

---

## 3. Channels

Each channel is opt-in. Enable only what you need.

### 📧 Email (SMTP)
Enable with `EMAIL_ENABLED=true`.

| Variable | Description |
|----------|-------------|
| `SMTP_HOST` | SMTP Server Host (e.g. `smtp.sendgrid.net`). |
| `SMTP_PORT` | SMTP Port (default `587`). |
| `SMTP_USERNAME` | SMTP Auth Username. |
| `SMTP_PASSWORD` | SMTP Auth Password. |
| `SMTP_FROM_EMAIL` | Sender email address. |
| `SMTP_FROM_NAME` | Sender display name. |

### 📱 SMS (Twilio)
Enable with `SMS_ENABLED=true`.

| Variable | Description |
|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Twilio Account SID. |
| `TWILIO_AUTH_TOKEN` | Twilio Auth Token. |
| `TWILIO_FROM_NUMBER` | Sending number (e.g., `+15550000000`). |

### 🔔 Push (Mobile)
Enable with `PUSH_ENABLED=true`.

**FCM (Android/iOS)**:
- `FCM_ENABLED=true`
- `FCM_PROJECT_ID`: Firebase Project ID.
- `FCM_CREDENTIALS`: **JSON Content** of the service account key (not file path).

**APNS (iOS)**:
- `APNS_ENABLED=true`
- `APNS_TEAM_ID`: Apple Team ID.
- `APNS_KEY_ID`: Key ID from Developer Portal.
- `APNS_BUNDLE_ID`: App Bundle ID (e.g., `com.example.app`).
- `APNS_KEY_FILE`: Path to the `.p8` key file.
- `APNS_PRODUCTION`: `true` for prod, `false` for sandbox.

### 💬 In-App (WebSocket)
Enable with `INAPP_ENABLED=true`.
- **Requires Redis** to handle Pub/Sub across multiple instances.
- **Scaling**: Increase `INAPP_WORKER_COUNT` (default `10`) for high-traffic WS handling.

---

## 4. Advanced Tuning

### Rate Limiting
Prevent spam or system overload.
- `RATE_LIMIT_ENABLED`: `true`/`false`.
- `RATE_LIMIT_MAX_PER_HOUR`: Max notifications per user/channel (default `100`).
- `RATE_LIMIT_WINDOW_SECONDS`: Time window (default `3600`).

### User Resolution
- `USER_SERVICE_URL`: URL to your user service (e.g. `http://user-service/users`). If set, the orchestrator will fetch user contact info from here automatically.
- `AUTH_ENABLED`: `true`/`false` (enforces JWT/API Key checks).
- `JWT_SECRET`: Secret to validate User JWTs.
- `API_KEYS`: Comma-separated allowlist for service-to-service calls (`key1:serviceA,key2:serviceB`).

### Observability
- `METRICS_ENABLED`: `true` (Exposes `/metrics`).
- `METRICS_PORT`: `9102`.
- `TRACING_ENABLED`: `true` (OpenTelemetry).
- `JAEGER_ENDPOINT`: URL to Jaeger collector.
