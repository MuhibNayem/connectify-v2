# Connectify Events Service

[![Pipeline Status](https://gitlab.com/spydotech-group/events-service/badges/main/pipeline.svg)](https://gitlab.com/spydotech-group/events-service/-/pipelines)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev/)

Standalone microservice for event management, recommendations, and RSVP tracking in the Connectify ecosystem.

## 🎯 Features

- **Event Management**: Create, update, and delete social events
- **Recommendations**: ML-powered event recommendations based on user interests and social graph
- **RSVP Tracking**: Real-time attendee management
- **Graph Integration**: Neo4j-powered social connections for personalized recommendations
- **Event-Driven Architecture**: Kafka-based async notifications

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- MongoDB
- Redis (cluster mode)
- Neo4j
- Kafka

### Installation

```bash
go get gitlab.com/spydotech-group/events-service@latest
```

### Running Locally

```bash
cp .env.example .env  # Configure your environment
go run cmd/api/main.go
```

### Docker

```bash
docker build -t events-service .
docker run -p 9096:9096 events-service
```

## 📡 API Endpoints

### gRPC (Port 9096)
- `CreateEvent`
- `GetEvent`
- `UpdateEvent`
- `DeleteEvent`
- `GetRecommendations`
- `RSVPEvent`

### Metrics (Port 9100)
- Prometheus metrics endpoint

## 🔄 Automatic Versioning

This repository uses GitLab CI/CD for automated releases:
- Tests run on every push
- New version tag created automatically on push to `main`
- Follows semantic versioning

## 🛠️ Development

### Running Tests

```bash
go test -v ./...
```

### Code Quality

```bash
go vet ./...
go fmt ./...
```

## 📦 Dependencies

- **Shared Entity**: `gitlab.com/spydotech-group/shared-entity`
- MongoDB Driver
- Redis
- Neo4j Driver
- Kafka
  
## 🏗️ Architecture

```
┌─────────────────┐                           ┌────────────────────┐
│  messaging-app  │                           │   events-service   │
│                 │                           │                    │
│  ┌────────────┐ │       gRPC (Port 9096)    │  ┌───────────────┐ │
│  │events      │ ├──────────────────────────>│  │  gRPC Server  │ │
│  │  client    │ │                           │  │               │ │
│  └────────────┘ │                           │  └───────┬───────┘ │
│                 │                           │          │         │
│  HTTP REST API  │                           │  ┌───────▼───────┐ │
│  (Port 8080)    │                           │  │   Service     │ │
│                 │                           │  │   Layer       │ │
└─────────────────┘                           │  └───────┬───────┘ │
                                              │          │         │
┌─────────────────┐                           │  ┌───────▼───────┐ │
│  shared-entity  │                           │  │ Repository    │ │
│                 │                           │  │   Layer       │ │
│  Proto Defs     │◄──────────────────────────┤  └───────┬───────┘ │
│  Models         │                           │          │         │
└─────────────────┘                           └──────────┼─────────┘
                                                         │
       ┌───────────────┬──────────────────┬──────────────▼─────┐
       │               │                  │                    │
┌──────▼──────┐  ┌─────▼─────┐      ┌─────▼───────┐      ┌─────▼─────┐
│   MongoDB   │  │   Redis   │      │    Neo4j    │      │   Kafka   │
│ (Events DB) │  │  (Cache)  │      │ (Graph DB)  │      │ (Streams) │
└─────────────┘  └───────────┘      └─────────────┘      └───────────┘
```
events-service/
├── cmd/api/          # Main entry point
├── config/           # Configuration management
├── internal/
│   ├── cache/        # Redis caching layer
│   ├── consumer/     # Kafka consumers
│   ├── controllers/  # gRPC handlers
│   ├── graph/        # Neo4j graph operations
│   ├── platform/     # Application bootstrap
│   ├── producer/     # Kafka producers
│   ├── repository/   # Data access layer
│   └── service/      # Business logic
└── proto/            # gRPC definitions
```

## 🔗 Related Repositories

- [shared-entity](https://gitlab.com/spydotech-group/shared-entity) - Shared models and utilities
- [messaging-app](https://gitlab.com/spydotech-group/messaging-app) - Main messaging application

## 📄 License

Proprietary - Connectify/SpydoTech Group

---

**Maintained by**: SpydoTech Group  
**Last Updated**: 2025-12-22
