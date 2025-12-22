# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy shared entity first (for go.mod replace)
COPY shared-entity ./shared-entity

# Copy feed-service go.mod
COPY feed-service/go.mod feed-service/go.sum ./feed-service/
WORKDIR /app/feed-service
RUN go mod download

# Copy source
COPY feed-service .

# Verify source files
RUN test -d cmd/api && test -f cmd/api/main.go || \
    (echo "Source files missing" && exit 1)

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o feed-service ./cmd/api

# Runtime stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/feed-service/feed-service .

RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup && \
    chown appuser:appgroup /app/feed-service
USER appuser

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 8080
EXPOSE 9090

ENTRYPOINT ["./feed-service"]
