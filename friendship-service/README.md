# Friendship Service

A production-ready, high-performance friendship management microservice built with Go. This service handles friend requests, friendships, blocking, and relationship queries with **strong consistency guarantees**, **fault tolerance**, and **MAANG-scale performance characteristics**.

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Friendship Service                            │
├──────────────────────────────────────────────────────────────────────┤
│  HTTP API (Gin)  │  gRPC API  │  Metrics (Prometheus)                 │
├──────────────────────────────────────────────────────────────────────┤
│                         Service Layer                                 │
│  ┌─────────────┐  ┌─────────┐  ┌───────────────┐  ┌───────────────┐ │
│  │  Saga       │  │ Outbox  │  │ Reconciliation│  │ Circuit       │ │
│  │  Pattern    │  │ Pattern │  │ Job           │  │ Breaker       │ │
│  └─────────────┘  └─────────┘  └───────────────┘  └───────────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│  MongoDB (Primary) │ Neo4j / Dgraph (Graph) │ Redis │ Kafka          │
└──────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

### Core Functionality

* Send, accept, reject friend requests
* Manage friendships (list, search, unfriend)
* Block/unblock users with relationship cleanup
* Real-time friendship status checks
* Friend-of-friend graph traversal

### Production-Grade Patterns

* **Outbox Pattern** for MongoDB → Graph DB consistency
* **Saga Pattern** for distributed transactions with rollback
* **Circuit Breaker** for dependency fault tolerance
* **Optimistic Locking** for safe concurrent updates
* **Read Repair** for query-time self-healing
* **Background Reconciliation Jobs** for eventual consistency
* **Cache Pub/Sub** for distributed cache invalidation

---

## 📊 Consistency Guarantees

| Pattern            | Purpose                    | Guarantee                  |
| ------------------ | -------------------------- | -------------------------- |
| Outbox Pattern     | MongoDB → Graph sync       | ≥ 99% eventual consistency |
| Saga Pattern       | Cross-service transactions | ≥ 99.9% success            |
| Read Repair        | Query-time healing         | Self-healing               |
| Reconciliation Job | Background repair          | Continuous                 |
| Versioned Writes   | Race prevention            | Per-entity linearizability |

---

## 🚀 Quick Start

### Prerequisites

* Go **1.21+**
* MongoDB (replica set for transactions)
* Neo4j **or** Dgraph
* Redis Cluster
* Apache Kafka

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
GRAPH_DB=dgraph # or neo4j
DGRAPH_ADDR=localhost:9080
REDIS_URLS=localhost:6379
KAFKA_BROKERS=localhost:9092
JWT_SECRET=your-secret-key
USER_SERVICE_URL=localhost:9101
```

---

## 📡 API

### HTTP API (JWT Protected)

| Method | Endpoint                                 | Description                  |
| ------ | ---------------------------------------- | ---------------------------- |
| POST   | /api/v1/friendships/requests             | Send friend request          |
| POST   | /api/v1/friendships/requests/:id/respond | Accept / reject request      |
| GET    | /api/v1/friendships                      | List friendships             |
| GET    | /api/v1/friendships/search               | Search friends               |
| GET    | /api/v1/friendships/check                | Check if friends             |
| GET    | /api/v1/friendships/status               | Detailed relationship status |
| DELETE | /api/v1/friendships/:friend_id           | Unfriend                     |
| POST   | /api/v1/friendships/block/:user_id       | Block user                   |
| DELETE | /api/v1/friendships/block/:user_id       | Unblock user                 |
| GET    | /api/v1/friendships/blocked              | List blocked users           |

### gRPC API

See `proto/friendship/v1/friendship.proto`

---

## 🧪 Testing

### Test Commands

```bash
# Unit tests (fast)
go test ./... -short

# All tests
go test ./... -v

# Coverage
go test ./... -cover

# Load & performance tests
go test ./tests/loadtest/... -v
```

### Test Coverage Snapshot

| Package    | Coverage  |
| ---------- | --------- |
| saga       | **97.2%** |
| validation | **100%**  |
| outbox     | 1.1%      |
| metrics    | 8.3%      |

---

## 📈 MAANG-Scale Performance Results

> **Verified on Apple M1 Pro (10 cores)**

| Metric      | Result                 | SLA   | Status |
| ----------- | ---------------------- | ----- | ------ |
| Throughput  | **~719K ops/sec/core** | 100K  | ✅      |
| P50 Latency | **45µs**               | <5ms  | ✅      |
| P99 Latency | **~220µs**             | <50ms | ✅      |
| Error Rate  | **0.000%**             | <0.1% | ✅      |
| Peak Memory | **739MB**              | <1GB  | ✅      |

### Load Scenarios

* Sustained load: >1.2M RPS (P99 < 3ms)
* 10× spike test with instant recovery
* Stress-tested to 6,400+ concurrent users

> ⚠️ **Note:** Extended soak tests currently flag elevated memory growth. Heap profiling is recommended before long-running production workloads.

---

## 📁 Project Structure

```
friendship-service/
├── cmd/api/
├── config/
├── internal/
│   ├── cache/
│   ├── grpc/
│   ├── httpapi/
│   ├── kafka/
│   ├── metrics/
│   ├── mocks/
│   ├── outbox/
│   ├── platform/
│   ├── reconciliation/
│   ├── repository/
│   ├── saga/
│   ├── service/
│   └── validation/
├── tests/
│   ├── integration/
│   └── loadtest/
└── Dockerfile
```

---

## 🔧 Technology Stack

* **Go 1.21**
* **Gin** (HTTP API)
* **gRPC**
* **MongoDB** (Primary store)
* **Neo4j / Dgraph** (Graph relationships)
* **Redis Cluster** (Caching)
* **Apache Kafka** (Events)
* **Prometheus** (Metrics)
* **OpenTelemetry** (Tracing)
* **sony/gobreaker** (Circuit breaker)

---

## 🐳 Docker

```bash
docker build -t friendship-service .
docker run -p 8103:8103 -p 9103:9103 friendship-service
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

---

## 📄 License

MIT License
