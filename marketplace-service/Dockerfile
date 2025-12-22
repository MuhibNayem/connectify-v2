# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o marketplace-service ./cmd/api

# Runtime stage
FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /app/marketplace-service /marketplace-service

USER nonroot:nonroot

EXPOSE 9097 9198

ENTRYPOINT ["/marketplace-service"]
