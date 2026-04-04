.PHONY: build test test-coverage lint docker-build docker-up docker-down clean

# Build the Go server (requires frontend/dist to exist for embed)
build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server

# Run all Go tests
test:
	go test ./...

# Run tests with coverage and enforce ≥90% gate
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linters
lint:
	go vet ./...

# Build the Docker image
docker-build:
	docker build -t lang-learn:latest .

# Start services via docker-compose
docker-up:
	docker compose up -d

# Stop services
docker-down:
	docker compose down

# Remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
