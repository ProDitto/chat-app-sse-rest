# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum to download dependencies first, leveraging Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application binary.
# CGO_ENABLED=0 creates a static binary, which is ideal for scratch/alpine images.
# -o /server places the output binary in the root directory named 'server'.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Stage 2: Create the final, lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /server .

# Copy the configuration file
COPY configs/config.yaml ./configs/

# Expose the port the application runs on
EXPOSE 8080

# Command to run the application
CMD ["./server"]

