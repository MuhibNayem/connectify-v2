.PHONY: build run test docker-build clean lint

BINARY_NAME=story-service
DOCKER_IMAGE=story-service

build:
	go build -o bin/$(BINARY_NAME) ./cmd/api

run:
	go run ./cmd/api

test:
	go test -v -race ./...

lint:
	go vet ./...

clean:
	go clean
	rm -rf bin/

docker-build:
	docker build -t $(DOCKER_IMAGE) .

# For convenient local development
dev:
	go run ./cmd/api
