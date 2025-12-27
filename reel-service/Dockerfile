FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

COPY reel-service/go.mod reel-service/go.sum ./reel-service/
WORKDIR /app/reel-service
RUN go mod download

COPY reel-service .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/reel-service ./cmd/main.go

FROM gcr.io/distroless/base-debian12

ENV HTTP_PORT=8086
ENV GRPC_PORT=9096

COPY --from=builder /bin/reel-service /bin/reel-service

EXPOSE 8086 9096

USER nonroot:nonroot

ENTRYPOINT ["/bin/reel-service"]
