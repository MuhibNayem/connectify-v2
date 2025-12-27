FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared-entity ./shared-entity

COPY storage-service/go.mod storage-service/go.sum ./storage-service/
WORKDIR /app/storage-service
RUN go mod download

COPY storage-service .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/storage-service ./cmd/main.go

FROM gcr.io/distroless/base-debian12

ENV HTTP_PORT=8087
ENV GRPC_PORT=9087

COPY --from=builder /bin/storage-service /bin/storage-service

EXPOSE 8087 9087

USER nonroot:nonroot

ENTRYPOINT ["/bin/storage-service"]
