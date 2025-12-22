# Connectify Shared Entity

[![Pipeline Status](https://gitlab.com/spydotech-group/shared-entity/badges/main/pipeline.svg)](https://gitlab.com/spydotech-group/shared-entity/-/pipelines)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev/)

Shared models, utilities, middleware, and proto definitions for Connectify microservices ecosystem.

```
                                  ┌──────────────────┐
                                  │                  │
                                  │   shared-entity  │
                                  │                  │
                                  └─────────┬────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
          ┌─────────▼────────┐    ┌─────────▼────────┐    ┌─────────▼────────┐
          │                  │    │                  │    │                  │
          │  messaging-app   │    │  events-service  │    │ marketplace-srv  │
          │                  │    │                  │    │                  │
          └──────────────────┘    └──────────────────┘    └──────────────────┘

Key Components:
1. **Proto Definitions**: gRPC service contracts (`.proto` files)
2. **Models**: Shared Go structs (MongoDB models)
3. **Utils**: Common helper functions
```

Shared models, utilities, middleware, and proto definitions for Connectify microservices ecosystem.

```
                                  ┌──────────────────┐
                                  │                  │
                                  │   shared-entity  │
                                  │                  │
                                  └─────────┬────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
          ┌─────────▼────────┐    ┌─────────▼────────┐    ┌─────────▼────────┐
          │                  │    │                  │    │                  │
          │  messaging-app   │    │  events-service  │    │ marketplace-srv  │
          │                  │    │                  │    │                  │
          └──────────────────┘    └──────────────────┘    └──────────────────┘

Key Components:
1. **Proto Definitions**: gRPC service contracts (`.proto` files)
2. **Models**: Shared Go structs (MongoDB models)
3. **Utils**: Common helper functions
```

## 📦 What's Inside

This module provides common functionality used across all Connectify microservices:

- **`/models`** - Shared data models (User, Event, Message, etc.)
- **`/proto`** - Protocol Buffer definitions and generated gRPC code
- **`/middleware`** - HTTP middleware (Auth, Rate Limiting)
- **`/redis`** - Redis cluster client wrapper
- **`/kafka`** - Kafka producer/consumer utilities
- **`/events`** - Event definitions for event-driven architecture
- **`/utils`** - Common utilities (JWT, validation, etc.)

## 🚀 Quick Start

### Installation

```bash
go get gitlab.com/spydotech-group/shared-entity@latest
```

### Usage

```go
import (
    "gitlab.com/spydotech-group/shared-entity/models"
    "gitlab.com/spydotech-group/shared-entity/middleware"
    "gitlab.com/spydotech-group/shared-entity/redis"
)

func main() {
    // Use shared models
    user := &models.User{
        Username: "john_doe",
        Email:    "john@example.com",
    }

    // Use middleware
    router.Use(middleware.AuthMiddleware(jwtSecret, redisClient))
    
    // Use Redis client
    redisClient := redis.NewClusterClient(redis.Config{
        RedisURLs: []string{"localhost:6379"},
    })
}
```

## 🔄 Automatic Versioning

This repository uses GitLab CI/CD for automated releases:

- **On every push to `main`**: Tests run automatically
- **After successful tests**: A new version tag is created (semantic versioning)
- **GitLab Release**: Automatically generated with release notes

### Version Format

Versions follow [Semantic Versioning](https://semver.org/):
- `v0.1.0` → `v0.1.1` (patch increment on each push)
- Breaking changes or features may require manual major/minor version bumps

## 🛠️ Development

### Local Development

When developing locally and testing with other microservices, use a `replace` directive:

```go
// In your service's go.mod
replace gitlab.com/spydotech-group/shared-entity => ../shared-entity
```

**⚠️ Important**: Remove the `replace` directive before committing!

### Running Tests

```bash
go test -v ./...
```

### Code Quality

```bash
go vet ./...
go fmt ./...
```

## 📚 Package Documentation

### Models
Common data structures used across services including User, Event, Message, Community, Story, and more.

### Middleware
- `AuthMiddleware` - JWT authentication with Redis blacklist
- `RateLimiter` - IP-based and user-based rate limiting
- `WSJwtAuthMiddleware` - WebSocket authentication

### Redis
Wrapper around `go-redis/v9` with cluster support, providing:
- Connection pooling
- Automatic failover
- Simplified API

### Kafka
Utilities for event-driven architecture:
- `DLQProducer` - Dead Letter Queue producer
- Event schemas for cross-service communication

### Proto
gRPC definitions for:
- Events service (`proto/events/v1`)
- Realtime service (`proto/realtime/v1`)

## 🤝 Contributing

1. Create a feature branch from `main`
2. Make your changes
3. Ensure tests pass: `go test ./...`
4. Create a merge request to `main`
5. After merge, a new version will be automatically released

## 📋 Dependencies

- Go 1.25+
- Redis (cluster mode)
- Kafka
- Protocol Buffers

## 📄 License

Proprietary - Connectify/SpydoTech Group

## 🔗 Related Repositories

- [events-service](https://gitlab.com/spydotech-group/events-service)
- [messaging-app](https://gitlab.com/spydotech-group/messaging-app)

---

**Maintained by**: SpydoTech Group  
**Last Updated**: 2025-12-22
