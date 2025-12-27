FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY ../shared-entity ../shared-entity

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /reel-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /reel-service .

EXPOSE 8086 9096

CMD ["./reel-service"]
