<![CDATA[<div align="center">
  <h1>🚀 Connectify V2</h1>
  <p><strong>A Hyperscale Social Networking Platform</strong></p>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go" />
    <img src="https://img.shields.io/badge/SvelteKit-2.0-FF3E00?logo=svelte&logoColor=white" alt="SvelteKit" />
    <img src="https://img.shields.io/badge/MongoDB-7.0-47A248?logo=mongodb&logoColor=white" alt="MongoDB" />
    <img src="https://img.shields.io/badge/Neo4j-5.0-4581C3?logo=neo4j&logoColor=white" alt="Neo4j" />
    <img src="https://img.shields.io/badge/Cassandra-4.1-1287B1?logo=apache-cassandra&logoColor=white" alt="Cassandra" />
    <img src="https://img.shields.io/badge/Kafka-3.6-231F20?logo=apache-kafka&logoColor=white" alt="Kafka" />
    <img src="https://img.shields.io/badge/Redis-7.2-DC382D?logo=redis&logoColor=white" alt="Redis" />
    <img src="https://img.shields.io/badge/Kubernetes-Ready-326CE5?logo=kubernetes&logoColor=white" alt="Kubernetes" />
  </p>
</div>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Key Features](#-key-features)
- [System Architecture](#-system-architecture)
- [Tech Stack](#-tech-stack)
- [Microservices Breakdown](#-microservices-breakdown)
- [Database Design Philosophy](#-database-design-philosophy)
- [Data Flow Patterns](#-data-flow-patterns)
- [Getting Started](#-getting-started)
- [License](#-license)

---

## 🌟 Overview

**Connectify V2** is a production-grade, distributed social networking platform engineered for hyperscale. It combines the best features of **Instagram** (Stories, Reels), **Facebook** (Events, Communities, Marketplace), and **WhatsApp** (End-to-End Encrypted Messaging) into a unified, modular architecture.

Built with **polyglot persistence**, **event-driven design**, and **FAANG-level optimizations**, Connectify is designed to handle millions of concurrent users with sub-100ms latency.

---

## ✨ Key Features

### 👤 User Management
- **Profile System** — Rich profiles with avatar, cover photo, bio, location
- **Privacy Controls** — Granular visibility settings (Public, Friends, Only Me)
- **Two-Factor Authentication** — Enhanced account security
- **End-to-End Encryption (E2EE)** — Client-side public/private key management
- **Presence System** — Real-time online/offline status with last seen

### 💬 Messaging (WhatsApp-Grade)
- **Direct Messages** — One-on-one private conversations
- **Group Chats** — Create and manage group conversations with roles
- **Message Reactions** — Emoji reactions on messages
- **Message Editing & Deletion** — Edit or soft-delete sent messages
- **Read Receipts** — Seen/delivered status indicators
- **Media Attachments** — Images, videos, voice messages via MinIO
- **Message Archiving** — Cassandra-backed infinite message history

### 📸 Stories & Reels (Instagram-Grade)
- **Ephemeral Stories** — 24-hour auto-expiring content
- **Privacy Controls** — Public, Friends, Custom, Friends-Except, Block Lists
- **View Tracking** — See who viewed your story
- **Story Reactions** — React with emojis
- **Reels** — Short-form video content

### 📰 Feed & Posts
- **Rich Posts** — Text, images, videos with hashtags
- **Comments & Replies** — Nested discussion threads
- **Reactions** — Emoji reactions on posts, comments, and replies
- **Photo Albums** — Organize media into collections
- **Hashtag Discovery** — Browse posts by hashtag

### 📅 Events
- **Event Creation** — Host public or private events
- **RSVP System** — Going, Interested, Not Going
- **Co-Host Management** — Add/remove event co-hosts
- **Event Recommendations** — AI-powered suggestions based on social graph
- **Trending Events** — Discover popular events
- **Event Categories** — Organized by type

### 🛍️ Marketplace
- **Product Listings** — Sell items with images, descriptions, pricing
- **Category Browser** — Navigate products by category
- **Search & Filter** — Advanced product search
- **Seller Profiles** — View seller information
- **View Tracking** — Track product popularity
- **Favorites** — Save products for later

### 👥 Communities
- **Community Creation** — Build interest-based groups
- **Post Moderation** — Admin approval workflows
- **Member Management** — Roles and permissions

### 🔍 Search & Discovery
- **Universal Search** — Search across users, posts, products, events
- **Friend Suggestions** — Neo4j-powered social recommendations

### 🔔 Notifications
- **Real-time Notifications** — Push and in-app alerts
- **Notification Preferences** — Granular control per category

---

## 🏗️ System Architecture

```mermaid
graph TD
    subgraph "Client Layer"
        Web[🌐 Web App<br/>SvelteKit + TailwindCSS]
        Mobile[📱 Mobile App<br/>React Native]
    end

    subgraph "Gateway Layer"
        LB[🔄 Load Balancer]
        Gateway[🚪 API Gateway]
    end

    Web --> LB
    Mobile --> LB
    LB --> Gateway

    subgraph "Microservices Layer"
        Gateway --> UserSvc[👤 User Service<br/>Auth, Profiles, Privacy]
        Gateway --> MsgSvc[� Messaging App<br/>Chat, Groups, Reels]
        Gateway --> StorySvc[� Story Service<br/>Ephemeral Content]
        Gateway --> FeedSvc[� Feed Service<br/>Posts, Albums]
        Gateway --> EventSvc[📅 Events Service<br/>RSVP, Recommendations]
        Gateway --> MktSvc[�️ Marketplace<br/>Products, Search]
    end

    subgraph "Data Layer"
        UserSvc --> MongoDB[(🍃 MongoDB)]
        UserSvc --> Neo4j[(🕸️ Neo4j)]
        UserSvc --> Redis[(🔴 Redis)]
        
        MsgSvc --> MongoDB
        MsgSvc --> Cassandra[(� Cassandra)]
        MsgSvc --> MinIO[(📦 MinIO)]
        
        StorySvc --> MongoDB
        StorySvc --> Redis
        
        FeedSvc --> MongoDB
        FeedSvc --> Neo4j
        
        EventSvc --> MongoDB
        EventSvc --> Neo4j
        
        MktSvc --> MongoDB
    end

    subgraph "Event Bus"
        UserSvc --> Kafka{📨 Kafka}
        MsgSvc --> Kafka
        StorySvc --> Kafka
        FeedSvc --> Kafka
        EventSvc --> Kafka
        MktSvc --> Kafka
        
        Kafka --> Workers[⚙️ Background Workers]
        Workers --> Neo4j
    end

    classDef svc fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    classDef db fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    classDef bus fill:#fff3e0,stroke:#ef6c00,stroke-width:2px
    
    class UserSvc,MsgSvc,StorySvc,FeedSvc,EventSvc,MktSvc svc
    class MongoDB,Neo4j,Redis,Cassandra,MinIO db
    class Kafka,Workers bus
```

---

## 🛠️ Tech Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | SvelteKit 2, TailwindCSS | Reactive UI with "Apple Liquid Glass" aesthetics |
| **Backend** | Go 1.25, gRPC, REST | High-performance microservices |
| **API Comm** | Protocol Buffers | Type-safe inter-service communication |
| **Document Store** | MongoDB 7 | User profiles, posts, events, products |
| **Graph Database** | Neo4j 5 | Social relationships, recommendations |
| **Time-Series Store** | Apache Cassandra | Chat message logs |
| **Object Storage** | MinIO | Media files (images, videos, attachments) |
| **Cache** | Redis Cluster | Session, relationship cache, rate limiting |
| **Message Queue** | Apache Kafka | Event-driven architecture |
| **Observability** | Prometheus, Jaeger | Metrics, distributed tracing |
| **Container** | Docker, Kubernetes | Deployment orchestration |

---

## � Microservices Breakdown

| Service | Port | Responsibilities |
|---------|------|------------------|
| **user-service** | 8080 / 9090 | Authentication, profiles, privacy, presence, E2EE keys |
| **messaging-app** | 8081 / 9091 | DMs, groups, reactions, archival, reels, communities |
| **story-service** | 8082 / 9092 | Ephemeral stories, view tracking, reactions |
| **feed-service** | 8083 / 9093 | Posts, comments, replies, albums, hashtags |
| **events-service** | 8084 / 9094 | Events, RSVPs, recommendations, co-hosts |
| **marketplace-service** | 8085 / 9095 | Products, categories, search, view counts |
| **shared-entity** | — | Proto definitions, shared models |

---

## 🧠 Database Design Philosophy

We employ **Polyglot Persistence** — using the right database for the right job:

| Database | Use Case | Why? |
|----------|----------|------|
| **MongoDB** | User profiles, Posts, Events, Products | Flexible schema, fast aggregations |
| **Neo4j** | Friendships, Follows, Recommendations | O(1) relationship traversal |
| **Cassandra** | Chat messages | High write throughput, linear scalability |
| **MinIO** | Media files | S3-compatible, decoupled blob storage |
| **Redis** | Session, Cache, Presence | Sub-ms latency, pub/sub |

### Key Optimizations
- **Denormalized Reads** — Seller/Category info embedded in Products
- **Read-Through Caching** — Relationship checks cached for 5 minutes
- **Async Event Processing** — View counts via Kafka + Batch Writes
- **Circuit Breakers** — Graceful degradation on service failures

---

## 🌊 Data Flow Patterns

### Synchronous Read
```
Client → API → Redis Cache (HIT?) → MongoDB → Response
```

### Async Write (Event-Driven)
```
Client → API → MongoDB (Write) → Kafka (Publish)
                                    ↓
                    Background Worker → Neo4j (Graph Sync)
```

### Messaging Flow
```
Client → Messaging App → MinIO (Upload Media)
                       → Cassandra (Store Message)
                       → WebSocket/Push (Notify Recipient)
```

---

## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose
- Go 1.25+
- Node.js 20+
- Make

### Quick Start
```bash
# Clone the repository
git clone https://github.com/MuhibNayem/connectify-v2.git
cd connectify-v2

# Start infrastructure
docker-compose up -d mongo redis kafka neo4j cassandra minio

# Configure environment
cp .env.sample .env

# Run all services
make run-all

# Start frontend
cd frontend && npm install && npm run dev
```

### Running Individual Services
```bash
cd user-service && make run
cd messaging-app && make run
cd story-service && make run
cd feed-service && make run
cd events-service && make run
cd marketplace-service && make run
```

---

## 📄 License

This project is proprietary software. All rights reserved.

---

<div align="center">
  <p><strong>Built for Scale. Designed for Millions.</strong></p>
  <p>⭐ Star this repo if you find it inspiring!</p>
</div>
]]>