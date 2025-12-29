FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared entity first for caching
COPY shared-entity /shared-entity

COPY community-service/go.mod community-service/go.sum ./community-service/
WORKDIR /app/community-service

# Mod replace directive needs access to shared-entity at ../shared-entity relative to go.mod
# Structure in build:
# /app (root)
# /shared-entity
# /app/community-service

# Wait, if I am in /app/community-service, then ../shared-entity is /app/shared-entity.
# It matches current layout.

RUN go mod download

COPY community-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/community-service/app .
COPY --from=builder /app/community-service/.env .

EXPOSE 9101

CMD ["./app"]
