FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

COPY story-service/go.mod story-service/go.sum ./story-service/
WORKDIR /app/story-service
RUN go mod download

COPY story-service .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/story-service ./cmd/main.go

# Final stage
FROM gcr.io/distroless/base-debian12

ENV GRPC_PORT=9097

COPY --from=builder /bin/story-service /bin/story-service

EXPOSE 9097

USER nonroot:nonroot

ENTRYPOINT ["/bin/story-service"]
