# Marketplace Service

[![Pipeline Status](https://gitlab.com/spydotech-group/marketplace-service/badges/main/pipeline.svg)](https://gitlab.com/spydotech-group/marketplace-service/-/pipelines)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev/)

Standalone gRPC microservice for marketplace/e-commerce functionality in the Connectify ecosystem.

## 🏗️ Architecture

```
┌─────────────────┐                           ┌──────────────────────┐
│  messaging-app  │                           │ marketplace-service  │
│                 │                           │                      │
│  ┌────────────┐ │       gRPC (Port 9097)    │  ┌────────────────┐  │
│  │marketplace │ ├──────────────────────────>│  │  gRPC Server   │  │
│  │  client    │ │                           │  │                │  │
│  └────────────┘ │                           │  └────────┬───────┘  │
│                 │                           │           │          │
│  HTTP REST API  │                           │  ┌────────▼───────┐  │
│  (Port 8080)    │                           │  │    Service     │  │
└─────────────────┘                           │  │    Layer       │  │
                                              │  └────────┬───────┘  │
                                              │           │          │
┌─────────────────┐                           │  ┌────────▼───────┐  │
│  shared-entity  │                           │  │  Repository    │  │
│                 │                           │  │    Layer       │  │
│  Proto Defs     │◄──────────────────────────┤  └────────┬───────┘  │
│  Models         │                           │           │          │
└─────────────────┘                           └───────────┼──────────┘
                                                          │
                                              ┌───────────▼─────────┐
                                              │      MongoDB        │
                                              │   (Products, Cat.)  │
                                              └─────────────────────┘
```

## 🎯 Key Concepts

### gRPC Communication
- **Server**: marketplace-service runs a gRPC server on port 9097
- **Client**: messaging-app has `marketplaceclient` package that connects via gRPC
- **Proto**: Definitions live in `shared-entity/proto/marketplace/v1/`
- **No Direct Import**: messaging-app never imports marketplace-service code directly

### Microservice Pattern
1. **Proto-First Design**: API contract defined in `.proto` files
2. **Shared Models**: Common data structures in `shared-entity`
3. **Service Independence**: Each service has its own database, logic, deployment
4. **gRPC Communication**: Fast, type-safe, bidirectional streaming support

## 📡 gRPC API

**Service**: `MarketplaceService`  
**Port**: 9097 (gRPC)

### Methods

```protobuf
service MarketplaceService {
  rpc CreateProduct(CreateProductRequest) returns (ProductResponse);
  rpc GetProduct(GetProductRequest) returns (ProductResponse);
  rpc UpdateProduct(UpdateProductRequest) returns (ProductResponse);
  rpc DeleteProduct(DeleteProductRequest) returns (Empty);
  rpc MarkProductSold(MarkProductSoldRequest) returns (Empty);
  rpc SearchProducts(SearchProductsRequest) returns (SearchProductsResponse);
  rpc GetCategories(Empty) returns (GetCategoriesResponse);
  rpc ToggleSaveProduct(ToggleSaveProductRequest) returns (ToggleSaveProductResponse);
  rpc GetSavedProducts(GetSavedProductsRequest) returns (SearchProductsResponse);
  rpc GetMarketplaceConversations(GetConversationsRequest) returns (GetConversationsResponse);
}
```

## 🚀 Quick Start

### Prerequisites
- Go 1.25+
- MongoDB
- Protocol Buffers compiler (protoc)

### Running Locally

```bash
# Configure environment
cp .env.example .env

# Build
go build -o marketplace-service ./cmd/api

# Run
./marketplace-service
```

### Docker

```bash
docker build -t marketplace-service .
docker run -p 9097:9097 marketplace-service
```

## 🔄 Development Workflow

### 1. Update Proto Definitions
```bash
cd ../shared-entity
vim proto/marketplace/v1/marketplace.proto

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/marketplace/v1/marketplace.proto
```

### 2. Implement gRPC Handler
```go
// internal/grpc/server.go
func (s *Server) CreateProduct(ctx context.Context, req *marketplacepb.CreateProductRequest) (*marketplacepb.ProductResponse, error) {
    // Implementation
}
```

### 3. Update Client (in messaging-app)
```go
// messaging-app/internal/marketplaceclient/
resp, err := client.CreateProduct(ctx, &pb.CreateProductRequest{...})
```

## 📦 Project Structure

```
marketplace-service/
├── cmd/api/
│   └── main.go              # gRPC server bootstrap
├── config/
│   └── config.go            # Configuration loader
├── internal/
│   ├── grpc/
│   │   └── server.go        # gRPC method implementations
│   ├── service/
│   │   └── marketplace_service.go  # Business logic
│   ├── repository/
│   │   └── marketplace_repo.go     # MongoDB data access
│   └── platform/
│       └── dependencies.go  # Dependency injection
├── .gitlab-ci.yml           # CI/CD pipeline
├── Dockerfile               # Container image
└── README.md               # This file
```

## 🔗 Dependencies

- **shared-entity**: `v0.0.2` - Proto definitions and shared models
- **MongoDB**: Products, categories storage
- **gRPC**: Service communication

## 🔐 Environment Variables

```bash
MONGO_URI=mongodb://root:example@mongodb1:27017,mongodb2:27017,mongodb3:27017/messaging_app?replicaSet=rs0&
CASSANDRA_HOSTS=cassandra
GRPC_PORT=9097
METRICS_PORT=9198
```

## 🧪 Testing

```bash
# Unit tests
go test -v ./...

# Integration test (requires MongoDB)
go test -v ./internal/repository/...

# gRPC client test
grpcurl -plaintext localhost:9097 list
```

## 🔄 CI/CD

- **Auto-versioning**: Patch version incremented on every push to `main`
- **Semantic versioning**: `v0.0.1`, `v0.0.2`, etc.
- **GitLab Pipeline**: Test → Release → Tag

## 📚 Related Services

- **messaging-app**: Main API gateway, contains marketplace client
- **events-service**: Similar gRPC microservice pattern
- **shared-entity**: Proto definitions and shared models

## 🤝 Integration Example

### Server Side (marketplace-service)
```go
// Implements gRPC server
type Server struct {
    marketplacepb.UnimplementedMarketplaceServiceServer
    service *service.MarketplaceService
}
```

### Client Side (messaging-app)
```go
// Connect to marketplace-service
client, err := grpc.Dial("localhost:9097", grpc.WithInsecure())
marketplaceClient := marketplacepb.NewMarketplaceServiceClient(client)

// Call methods
resp, err := marketplaceClient.GetProduct(ctx, &marketplacepb.GetProductRequest{
    ProductId: "507f1f77bcf86cd799439011",
    ViewerId:  "507f191e810c19729de860ea",
})
```

## 📄 License

Proprietary - Connectify/SpydoTech Group

---

**Maintained by**: SpydoTech Group  
**Last Updated**: 2025-12-22
