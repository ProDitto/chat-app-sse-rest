# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /server .
COPY configs/config.yaml ./configs/
EXPOSE 8080
CMD ["/app/server"]

