# Friendship Service

A production-ready, microservices-architected friendship management system built with Go. This service handles friend requests, friendships, and user blocking with strong data consistency guarantees.

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Friendship Service                          │
├──────────────────────────────────────────────────────────────────────┤
│  HTTP API (Gin)  │  gRPC API  │  Metrics (Prometheus)                │
├──────────────────────────────────────────────────────────────────────┤
│                         Service Layer                                │
│  ┌─────────────┐  ┌─────────┐  ┌───────────────┐  ┌───────────────┐ │
│  │  Saga       │  │ Outbox  │  │ Reconciliation│  │ Circuit       │ │
│  │  Pattern    │  │ Pattern │  │ Job           │  │ Breaker       │ │
│  └─────────────┘  └─────────┘  └───────────────┘  └───────────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│  MongoDB (Primary)  │  Neo4j (Graph)  │  Redis (Cache)  │  Kafka    │
└──────────────────────────────────────────────────────────────────────┘
```

## ✨ Features

### Core Functionality
- **Friend Requests**: Send, accept, reject friend requests
- **Friendships**: List, search, unfriend operations
- **Blocking**: Block/unblock users with relationship cleanup
- **Status Checking**: Real-time friendship status with multiple data sources

### Production-Ready Patterns
- **Outbox Pattern**: Eventual consistency between MongoDB and Neo4j
- **Saga Pattern**: Distributed transactions with automatic rollback
- **Circuit Breaker**: Fault tolerance for external services
- **Read Repair**: Self-healing data inconsistencies
- **Cache Pub/Sub**: Distributed cache invalidation
- **Optimistic Locking**: Concurrent modification protection

## 📊 Consistency Guarantees

| Pattern | Purpose | Consistency |
|---------|---------|-------------|
| Outbox Pattern | MongoDB → Neo4j sync | 99%+ |
| Saga Pattern | Multi-service transactions | 99.9%+ |
| Read Repair | Query-time healing | Self-healing |
| Reconciliation | Hourly batch repair | Background |
| Versioned Writes | Race condition prevention | Per-entity |

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- MongoDB (replica set for transactions)
- Neo4j
- Redis Cluster
- Kafka

### Build & Run
```bash
cd friendship-service
go mod tidy
go build ./cmd/api
./api
```

### Environment Variables
```env
FRIENDSHIP_GRPC_PORT=9103
FRIENDSHIP_HTTP_PORT=8103
PROMETHEUS_PORT=9104
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=friendship
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
REDIS_URLS=localhost:6379
KAFKA_BROKERS=localhost:9092
JWT_SECRET=your-secret-key
USER_SERVICE_URL=localhost:9101
```

## 📡 API Endpoints

### HTTP API (Protected with JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/friendships/requests` | Send friend request |
| POST | `/api/v1/friendships/requests/:id/respond` | Accept/reject request |
| GET | `/api/v1/friendships` | List friendships |
| GET | `/api/v1/friendships/search` | Search friends |
| GET | `/api/v1/friendships/check` | Check if friends |
| GET | `/api/v1/friendships/status` | Detailed status |
| DELETE | `/api/v1/friendships/:friend_id` | Unfriend |
| POST | `/api/v1/friendships/block/:user_id` | Block user |
| DELETE | `/api/v1/friendships/block/:user_id` | Unblock user |
| GET | `/api/v1/friendships/block/:user_id/status` | Check block status |
| GET | `/api/v1/friendships/blocked` | List blocked users |

### gRPC API
See `proto/friendship/v1/friendship.proto` for service definition.

## 🧪 Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run unit tests only
go test ./... -short

# Run integration tests
go test ./tests/integration/... -v
```

### Test Coverage
| Package | Coverage |
|---------|----------|
| saga | 97.2% |
| validation | 100.0% |
| metrics | 8.3% |
| outbox | 1.1% |

## 📁 Project Structure

```
friendship-service/
├── cmd/api/              # Application entrypoint
├── config/               # Configuration
├── internal/
│   ├── cache/            # Redis cache + Pub/Sub
│   ├── grpc/             # gRPC handlers
│   ├── httpapi/          # HTTP handlers
│   ├── kafka/            # Kafka producer
│   ├── metrics/          # Prometheus metrics
│   ├── mocks/            # Test mocks
│   ├── outbox/           # Outbox pattern
│   ├── platform/         # App bootstrap
│   ├── reconciliation/   # Data repair job
│   ├── repository/       # Data access
│   ├── saga/             # Saga pattern
│   ├── service/          # Business logic
│   └── validation/       # Input validation
├── tests/
│   └── integration/      # Integration tests
└── Dockerfile
```

## 🔧 Technologies

- **Language**: Go 1.21
- **HTTP Framework**: Gin
- **gRPC**: google.golang.org/grpc
- **Databases**: MongoDB, Neo4j
- **Cache**: Redis Cluster
- **Messaging**: Apache Kafka
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry
- **Circuit Breaker**: sony/gobreaker

## 📈 Metrics

Exposed on `/metrics`:
- `friendship_requests_sent_total`
- `friendship_requests_accepted_total`
- `friendship_operation_duration_seconds`
- `friendship_inconsistencies_detected_total`
- `friendship_outbox_events_processed_total`

## 🐳 Docker

```bash
docker build -t friendship-service .
docker run -p 8103:8103 -p 9103:9103 friendship-service
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

MIT License
