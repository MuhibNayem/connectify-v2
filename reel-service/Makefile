.PHONY: run build test clean proto

run:
	go run cmd/main.go

build:
	go build -o bin/reel-service cmd/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/

proto:
	cd ../shared-entity && protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/reel/v1/reel.proto
