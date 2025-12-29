FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

COPY friendship-service/go.mod friendship-service/go.sum ./friendship-service/
WORKDIR /app/friendship-service
RUN go mod download

COPY friendship-service .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/friendship-service ./cmd/api

# Final stage
FROM gcr.io/distroless/base-debian12

ENV FRIENDSHIP_GRPC_PORT=9103 \
    FRIENDSHIP_HTTP_PORT=8103 \
    PROMETHEUS_PORT=9104

COPY --from=builder /bin/friendship-service /bin/friendship-service

EXPOSE 8103 9103 9104

USER nonroot:nonroot

ENTRYPOINT ["/bin/friendship-service"]
