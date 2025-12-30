# Universal Notification Service - Environment Configuration

## Complete Configuration Guide

All settings are configurable via environment variables. NO hardcoded values!

### 🗄️ Storage Configuration

```bash
# Storage Type - Choose ANY database you want
STORAGE_TYPE=memory          # Development/testing
STORAGE_TYPE=mongodb         # Production - MongoDB
STORAGE_TYPE=postgres        # Production - PostgreSQL
STORAGE_TYPE=mysql           # Add your own adapter
STORAGE_TYPE=dynamodb        # Add your own adapter
STORAGE_TYPE=cassandra       # Add your own adapter

# MongoDB Settings
STORAGE_URI=mongodb://localhost:27017
STORAGE_DATABASE=notifications
STORAGE_COLLECTION=notifications

# PostgreSQL Settings
STORAGE_URI=postgresql://user:pass@localhost:5432/notifications?sslmode=disable

# Connection Pooling
STORAGE_MAX_CONNECTIONS=25
STORAGE_MAX_IDLE=5
STORAGE_CONN_LIFETIME=300s
```

### 📨 Queue Configuration

```bash
# Queue Type - Choose ANY message queue
QUEUE_TYPE=memory            # Development/testing
QUEUE_TYPE=kafka             # Production - Apache Kafka
QUEUE_TYPE=rabbitmq          # Add your own adapter
QUEUE_TYPE=sqs               # Add your own adapter
QUEUE_TYPE=redis             # Add your own adapter

# Kafka Settings
QUEUE_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
QUEUE_TOPIC=notifications
QUEUE_GROUP_ID=notification-service
QUEUE_BATCH_SIZE=1000
QUEUE_BATCH_TIMEOUT=10ms

# Consumer Settings
QUEUE_CONSUMER_COUNT=10
QUEUE_MAX_RETRY=3
```

### 🚀 Server Configuration

```bash
# HTTP API Server
HTTP_PORT=8090
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_IDLE_TIMEOUT=60s
HTTP_MAX_HEADER_BYTES=1048576

# gRPC Server (Future)
GRPC_PORT=50060
GRPC_MAX_CONN_AGE=30m

# Environment
ENVIRONMENT=production       # production, staging, development
```

### 💾 Redis Configuration

```bash
# Redis Connection
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s
```

### 🛡️ Rate Limiting

```bash
# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_MAX_PER_HOUR=1000
RATE_LIMIT_WINDOW_SECONDS=3600
RATE_LIMIT_BURST=100
```

### 📺 Channel Configuration

```bash
# In-App Channel (WebSocket)
INAPP_ENABLED=true
INAPP_WORKER_COUNT=20
INAPP_QUEUE_SIZE=10000
INAPP_MAX_CONNECTIONS_PER_USER=5

# Push Notifications
PUSH_ENABLED=true
PUSH_WORKER_COUNT=30
PUSH_QUEUE_SIZE=20000

# FCM (Firebase Cloud Messaging)
FCM_ENABLED=true
FCM_PROJECT_ID=your-project-id
FCM_CREDENTIALS_FILE=/path/to/credentials.json

# APNS (Apple Push Notification Service)
APNS_ENABLED=true
APNS_KEY_ID=your-key-id
APNS_TEAM_ID=your-team-id
APNS_BUNDLE_ID=com.yourapp
APNS_KEY_FILE=/path/to/AuthKey.p8
APNS_PRODUCTION=true

# Email Channel
EMAIL_ENABLED=true
EMAIL_WORKER_COUNT=15
EMAIL_QUEUE_SIZE=5000
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=notifications@yourapp.com
SMTP_FROM_NAME=YourApp Notifications
SMTP_USE_TLS=true

# SMS Channel
SMS_ENABLED=true
SMS_WORKER_COUNT=10
SMS_QUEUE_SIZE=3000
TWILIO_ACCOUNT_SID=your-account-sid
TWILIO_AUTH_TOKEN=your-auth-token
TWILIO_FROM_NUMBER=+1234567890

# Webhook Channel
WEBHOOK_ENABLED=true
WEBHOOK_WORKER_COUNT=15
WEBHOOK_QUEUE_SIZE=5000
WEBHOOK_TIMEOUT=30s
WEBHOOK_MAX_RETRY=3
```

### 📊 Observability

```bash
# Logging
LOG_LEVEL=info               # debug, info, warn, error
LOG_FORMAT=json              # json, console
LOG_CALLER=true
LOG_STACKTRACE=false

# Metrics (Prometheus)
METRICS_ENABLED=true
METRICS_PORT=9102
METRICS_PATH=/metrics

# Tracing (OpenTelemetry)
TRACING_ENABLED=true
TRACING_EXPORTER=jaeger      # jaeger, zipkin, otlp
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
TRACING_SAMPLE_RATE=0.1

# Health Checks
HEALTH_CHECK_INTERVAL=30s
HEALTH_CHECK_TIMEOUT=5s
```

### 🔒 Security

```bash
# Authentication
AUTH_ENABLED=true
AUTH_TYPE=jwt                # jwt, api_key, oauth2
JWT_SECRET=your-secret-key
JWT_ISSUER=notification-service
JWT_EXPIRATION=24h

# API Keys
API_KEY_HEADER=X-API-Key
API_KEYS=key1,key2,key3      # Comma-separated

# CORS
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,X-API-Key
CORS_MAX_AGE=3600
```

### 🧹 Cleanup & Retention

```bash
# Retention Policies
RETENTION_ENABLED=true
RETENTION_DEFAULT_DAYS=90
RETENTION_READ_DAYS=30
RETENTION_CLEANUP_INTERVAL=1h

# Auto-cleanup
AUTO_CLEANUP_ENABLED=true
AUTO_CLEANUP_BATCH_SIZE=1000
```

### 🔧 Circuit Breaker

```bash
# Circuit Breaker Settings
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_TIMEOUT=60s
CIRCUIT_BREAKER_HALF_OPEN_REQUESTS=3
```

### 🎯 Feature Flags

```bash
# Feature Toggles
FEATURE_BATCH_OPERATIONS=true
FEATURE_SSE_STREAMING=true
FEATURE_WEBHOOKS=true
FEATURE_TEMPLATES=true
FEATURE_PRIORITY_QUEUE=true
```

## 📝 Usage Examples

### Development Setup
```bash
export STORAGE_TYPE=memory
export QUEUE_TYPE=memory
export LOG_LEVEL=debug
export LOG_FORMAT=console
./notification-service
```

### Production Setup (Docker)
```bash
docker run -e STORAGE_TYPE=mongodb \
  -e STORAGE_URI=mongodb://mongo:27017 \
  -e QUEUE_TYPE=kafka \
  -e QUEUE_BROKERS=kafka:9092 \
  -e REDIS_ADDR=redis:6379 \
  -e RATE_LIMIT_ENABLED=true \
  -e METRICS_ENABLED=true \
  -e LOG_FORMAT=json \
  notification-service
```

### Production Setup (Kubernetes)
Use ConfigMaps and Secrets:
```yaml
env:
  - name: STORAGE_TYPE
    value: "mongodb"
  - name: STORAGE_URI
    valueFrom:
      secretKeyRef:
        name: notification-secrets
        key: mongo-uri
```

## 🔄 Hot Reload

Some configurations support hot reload without restart:
- Logging level
- Rate limit values
- Feature flags
- Circuit breaker thresholds

Others require service restart:
- Storage/Queue type
- Connection settings
- Server ports

---

**100% Configurable  | Zero Hardcoded Values | Production Ready**
