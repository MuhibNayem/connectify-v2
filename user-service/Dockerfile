FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy shared-entity first (due to replace directive)
# Copy shared-entity first (due to replace directive)
COPY shared-entity ./shared-entity

# Copy user-service go.mod and go.sum
COPY user-service/go.mod user-service/go.sum* ./user-service/

WORKDIR /app/user-service
RUN go mod download

# Copy source code
COPY user-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service cmd/api/main.go

# Runtime stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/user-service/user-service .
COPY --from=builder /app/user-service/.env* .

EXPOSE 8083

CMD ["./user-service"]
