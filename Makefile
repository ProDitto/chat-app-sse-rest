.PHONY: run build docker-build docker-up docker-down

run:
	go run ./cmd/server

build:
	go build -o ./bin/server ./cmd/server

docker-build:
	docker build -t sse-chat-server .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

