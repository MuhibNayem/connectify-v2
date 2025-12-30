FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/notification-service ./cmd/server/main.go

FROM gcr.io/distroless/base-debian12

ENV SERVER_PORT=8090
ENV GRPC_PORT=50060

COPY --from=builder /bin/notification-service /bin/notification-service

EXPOSE 8090 50060 9102

USER nonroot:nonroot

ENTRYPOINT ["/bin/notification-service"]
