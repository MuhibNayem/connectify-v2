# Events Integration Layer

This package contains the **Anti-Corruption Layer (ACL)** and **integration adapters** for the Events Microservice.

## Purpose

The Events Service is a separate bounded context from the main Messaging App. However, it needs access to some data owned by other services, specifically:
- **Users**: To display event creators and attendees.
- **Friendships**: To recommend events based on social graph.

## The "Shared Database" Pattern

Currently, we are using the **Shared Database** pattern as a pragmatic first step towards microservices.
- The Events Service connects to the same physical MongoDB cluster as the Gateway.
- However, it **does not import** the `User` or `Friendship` models from the Gateway codebase.
- Instead, it maintains its own **local, read-only definitions** of these models in this package (`models.go`).

## How to use

1. Initialize these repositories in `bootstrap.go`.
2. Inject them into your Event Services.
3. This allows you to split this code into a separate Git repository (`events-service`) without any compilation errors, as it has no dependencies on `messaging-app/internal/...`.
