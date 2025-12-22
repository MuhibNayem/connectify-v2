# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

# Copy go mod files
COPY marketplace-service/go.mod marketplace-service/go.sum ./marketplace-service/
WORKDIR /app/marketplace-service
RUN go mod download

# Copy source code
COPY marketplace-service .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o marketplace-service ./cmd/api

# Runtime stage
FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /app/marketplace-service/marketplace-service /marketplace-service

USER nonroot:nonroot

EXPOSE 9097 9198

ENTRYPOINT ["/marketplace-service"]
