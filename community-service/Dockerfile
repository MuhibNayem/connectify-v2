FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

COPY community-service/go.mod community-service/go.sum ./community-service/
WORKDIR /app/community-service
RUN go mod download

COPY community-service .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/community-service ./cmd/main.go

# Final stage
FROM gcr.io/distroless/base-debian12

ENV COMMUNITY_GRPC_PORT=9101

COPY --from=builder /bin/community-service /bin/community-service

EXPOSE 9101

USER nonroot:nonroot

ENTRYPOINT ["/bin/community-service"]
