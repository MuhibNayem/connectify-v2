# Reel Service

Short-form video content microservice for Connectify, handling reels creation, feeds, reactions, and comments.

## Features

- **Reel CRUD** — Create, read, delete reels
- **Privacy-Filtered Feed** — Public reels + friends-only reels
- **View Counting** — Async via Kafka events
- **Reactions** — Toggle-style reactions (like/love/etc)
- **Comments & Replies** — Nested discussions with mentions
- **gRPC API** — Service-to-service communication

## Tech Stack

- **Go 1.25** — High-performance backend
- **MongoDB** — Reel storage
- **Redis** — Friend list caching
- **Kafka** — Event streaming
- **gRPC** — Inter-service calls to user-service

## Ports

| Protocol | Port |
|----------|------|
| HTTP | 8086 |
| gRPC | 9096 |

## Quick Start

```bash
# Copy environment
cp .env.example .env

# Run locally
make run

# Build binary
make build

# Run tests
make test
```

## API Endpoints

### HTTP API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/reels` | Create reel |
| GET | `/api/v1/reels/feed` | Get feed |
| GET | `/api/v1/reels/:id` | Get reel |
| DELETE | `/api/v1/reels/:id` | Delete reel |
| POST | `/api/v1/reels/:id/view` | Record view |
| POST | `/api/v1/reels/:id/react` | React to reel |
| GET | `/api/v1/reels/:id/comments` | List comments |
| POST | `/api/v1/reels/:id/comments` | Add comment |
| GET | `/api/v1/users/:id/reels` | Get user's reels |

### gRPC API

- `ReelService.GetReel`
- `ReelService.GetUserReels`
- `ReelService.GetReelsFeed`

## Dependencies

- **user-service** (gRPC) — Author info, friend lists, mentions
- **MongoDB** — Primary data store
- **Redis** — Caching
- **Kafka** — Event publishing
