FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/events-service ./cmd/api

# Final stage
FROM gcr.io/distroless/base-debian12

ENV EVENTS_GRPC_PORT=9096 \
    EVENTS_METRICS_PORT=9100

COPY --from=builder /bin/events-service /bin/events-service

EXPOSE 9096 9100

USER nonroot:nonroot

ENTRYPOINT ["/bin/events-service"]
