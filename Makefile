.PHONY: build run test clean docker-up docker-down

build:
	go build -o bin/harbor ./cmd/harbor

run: build
	./bin/harbor

test:
	go test ./... -v

clean:
	rm -rf bin/

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f harbor

lint:
	golangci-lint run ./...
