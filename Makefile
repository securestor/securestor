.PHONY: help setup run dev test docker-up docker-down docker-logs clean

help: ## Show this help message
	@echo "📚 SecureStor Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup: ## Run initial setup
	@./scripts/setup.sh

run: ## Run the API server
	@echo "🚀 Starting SecureStor API..."
	@go run cmd/api/main.go

dev: ## Run with hot reload (requires air)
	@echo "🔥 Starting SecureStor API with hot reload..."
	@air

build: ## Build the application binary
	@echo "🔨 Building SecureStor..."
	@go build -o bin/securestor cmd/api/main.go
	@echo "✅ Build complete: bin/securestor"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

docker-up: ## Start Docker containers
	@echo "🐳 Starting Docker containers..."
	@docker compose up -d
	@echo "✅ Containers started"

docker-down: ## Stop Docker containers
	@echo "🛑 Stopping Docker containers..."
	@docker compose down
	@echo "✅ Containers stopped"

docker-logs: ## View Docker logs
	@docker compose logs -f

docker-rebuild: ## Rebuild and restart Docker containers
	@echo "🔨 Rebuilding Docker containers..."
	@docker compose down
	@docker compose up -d --build
	@echo "✅ Containers rebuilt and started"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

migrate: ## Run database migrations
	@echo "🔄 Running migrations..."
	@go run cmd/api/main.go migrate
	@echo "✅ Migrations complete"

.DEFAULT_GOAL := help
