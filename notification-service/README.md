# Universal Notification Service

A robust, event-driven notification orchestration engine designed for high-scale microservices architectures.

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/MuhibNayem/connectify-v2/notification-service)](https://goreportcard.com/report/github.com/MuhibNayem/connectify-v2/notification-service)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## 🚀 Overview

This service standardizes how notifications (Email, SMS, Push, In-App) are dispatched across your entire platform. Instead of every microservice implementing its own email client or Twilio integration, they simply push an event to this service, which handles:

- **Orchestration**: Priority routing, templating, and user preference checks.
- **Reliability**: Retries, Circuit Breaking, and Dead Letter Queues (DLQ).
- **Scalability**: Asynchronous processing with Kafka/RabbitMQ and worker pools.

## ✨ Key Features

- **Multi-Channel Support**: Email (SMTP), SMS (Twilio), Push (FCM/APNS), In-App (WebSocket).
- **Priority Queues**: Separate "Fast Lane" for critical alerts (e.g., OTPs).
- **Idempotency**: Guarantees exactly-once processing using Redis.
- **Pluggable Architecture**: Easily swap storage (Postgres/Mongo) and queues (Kafka/RabbitMQ).
- **Observability**: Built-in Prometheus metrics and distributed tracing.

## 📖 Documentation

For detailed integration instructions, API references, and configuration options, please read the **[User Guide](USERGUIDE.md)**.

## ⚡ Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)

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

3. **Send a test notification:**
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

## 🛠️ Tech Stack

- **Language**: Go
- **Messaging**: Kafka, RabbitMQ
- **Storage**: PostgreSQL, MongoDB
- **Cache**: Redis
- **Observability**: Prometheus, Jaeger

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
