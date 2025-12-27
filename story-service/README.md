# Story Service

[![Pipeline Status](https://gitlab.com/spydotech-group/story-service/badges/main/pipeline.svg)](https://gitlab.com/spydotech-group/story-service/-/pipelines)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev/)

Standalone gRPC and HTTP microservice for managing ephemeral stories (Snapchat/Instagram style) in the Connectify ecosystem.

## 🏗️ Architecture

```
┌─────────────────┐                           ┌──────────────────────┐
│  messaging-app  │                           │    story-service     │
│                 │                           │                      │
│  ┌────────────┐ │       gRPC/HTTP           │  ┌────────────────┐  │
│  │story client│ ├──────────────────────────>│  │  gRPC Server   │  │
│  └────────────┘ │                           │  │  HTTP API      │  │
│                 │                           │  └────────┬───────┘  │
│                 │                           │           │          │
│                 │                           │  ┌────────▼───────┐  │
│                 │                           │  │    Service     │  │
│                 │                           │  │    Layer       │  │
│                 │                           │  └────────┬───────┘  │
│                 │                           │           │          │
│                 │                           │  ┌────────▼───────┐  │
│                 │                           │  │  Repository    │  │
│                 │                           │  │    Layer       │  │
│                 │                           │  └────────┬───────┘  │
│                 │                           │           │          │
│                 │                           └───────────┼──────────┘
│                 │                                       │
│                 │                           ┌───────────▼─────────┐
│                 │                           │      MongoDB        │
│                 │                           │      (Stories)      │
│                 │                           └─────────────────────┘
└─────────────────┘
```

## 🎯 Key Features

- **Ephemeral Stories**: Create stories that expire after 24 hours.
- **Privacy Controls**: Granular visibility settings (Public, Friends, Custom, Block Lists).
- **View Tracking**: Track who viewed your story with real-time updates.
- **Reactions**: React to stories with emojis.
- **Resilience**: Circuit breakers for external service dependencies.
- **Event-Driven**: Asynchronous event publishing via Kafka.

## 📡 API

**HTTP Port**: 8082
**gRPC Port**: 9092 (if configured)

### Endpoints

- `POST /stories`: Create a new story
- `GET /stories/feed`: Get story feed from friends
- `GET /stories/my`: Get current user's active stories
- `POST /stories/{id}/view`: Mark a story as viewed
- `POST /stories/{id}/react`: React to a story
- `DELETE /stories/{id}`: Delete a story

## 🚀 Quick Start

### Prerequisites
- Go 1.25+
- MongoDB
- Kafka (optional, for events)

### Running Locally

```bash
# Configure environment
cp .env.example .env

# Run dependencies (if not standard)
# docker-compose up -d mongo kafka

# Build and Run
make run
```

### Docker

```bash
docker build -t story-service .
docker run -p 8082:8082 story-service
```

## 📦 Project Structure

```
story-service/
├── cmd/api/
│   └── main.go              # Service entrypoint
├── config/
│   └── config.go            # Configuration loader
├── internal/
│   ├── service/             # Business logic (story_service.go)
│   ├── repository/          # Data access (MongoDB)
│   ├── httpapi/             # HTTP handlers
│   ├── grpc/                # gRPC server (if applicable)
│   ├── metrics/             # Prometheus metrics
│   └── producer/            # Kafka producer
├── Makefile                 # Developer commands
├── .gitlab-ci.yml           # CI/CD pipeline
└── README.md                # This file
```

## 🔗 Dependencies

- **shared-entity**: Proto definitions and shared models
- **MongoDB**: Storage
- **Kafka**: Event streaming

## 🧪 Testing

```bash
# Unit tests
make test
```

## 📄 License

Proprietary - Connectify/SpydoTech Group
