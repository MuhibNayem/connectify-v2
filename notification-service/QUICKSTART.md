# Quick Start Guide

## 🚀 Get Started in 60 Seconds

### Option 1: Run with Go

```bash
cd notification-service

# Install dependencies
go mod download

# Build and run
go run cmd/server/main.go
```

Server starts at: `http://localhost:8090`

### Option 2: Docker

```bash
cd notification-service

# Build and run with docker-compose
docker-compose up --build
```

### Option 3: Pre-built Binary

```bash
cd notification-service

# Build
go build -o notification-service ./cmd/server

# Run
./notification-service
```

## ✅ Verify It's Working

```bash
# Health check
curl http://localhost:8090/health

# Expected response:
# {"status":"healthy","service":"notification-service"}
```

## 📨 Send Your First Notification

```bash
curl -X POST http://localhost:8090/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_id": "user_123",
    "type": "welcome",
    "title": "Welcome!",
    "body": "Thanks for trying the Universal Notification Service",
    "priority": "normal"
  }'
```

## 📖 List Notifications

```bash
curl "http://localhost:8090/v1/notifications?recipient_id=user_123"
```

## 📊 Check Unread Count

```bash
curl "http://localhost:8090/v1/notifications/unread-count?recipient_id=user_123"
```

## ✔️ Mark as Read

```bash
# Get the notification ID from the list above, then:
curl -X PATCH http://localhost:8090/v1/notifications/NOTIFICATION_ID \
  -H "Content-Type: application/json" \
  -d '{"read": true}'
```

## 🎯 Batch Mark as Read

```bash
curl -X POST http://localhost:8090/v1/notifications/batch/mark-read \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_id": "user_123",
    "mark_all_as_read": true
  }'
```

## 🔌 Next Steps

1. **Add MongoDB Storage**: Change `STORAGE_TYPE=mongodb` in docker-compose.yml
2. **Add Kafka Queue**: Change `QUEUE_TYPE=kafka` in docker-compose.yml
3. **Enable Channels**: Set `PUSH_ENABLED=true` to enable push notifications
4. **Integrate Your Auth**: Implement the `AuthenticationAdapter` interface
5. **Add User Preferences**: Implement the `UserPreferenceAdapter` interface

See the [README.md](README.md) for complete documentation.

## 🐛 Troubleshooting

**Port already in use?**
```bash
# Change the port
export HTTP_PORT=8091
./notification-service
```

**Want to see debug logs?**
```bash
export LOG_LEVEL=debug
./notification-service
```

## 📚 Full API Documentation

See [OpenAPI Specification](pkg/api/v1/openapi.yaml) for complete API docs.

You can also import this into Postman or use Swagger UI!
