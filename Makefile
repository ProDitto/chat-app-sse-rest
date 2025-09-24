.PHONY: run build docker-build docker-up docker-down

# Go variables
BINARY_NAME=server
BINARY_PATH=./bin/$(BINARY_NAME)

# Run the Go application locally
run:
	go run ./cmd/server/main.go

# Build the Go binary for the local OS
build:
	mkdir -p ./bin
	go build -o $(BINARY_PATH) ./cmd/server/main.go

# Build the Docker image for the application
docker-build:
	docker build -t sse-chat-app .

# Start the application and its dependencies using Docker Compose in detached mode
docker-up:
	docker-compose up -d

# Stop and remove the Docker containers, networks, and volumes
docker-down:
	docker-compose down
