# Makefile for Purchase Approval System

.PHONY: help deps temporal-up temporal-down worker web test clean

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Download Go dependencies
	@echo "Downloading Go dependencies..."
	go mod tidy
	go mod download

temporal-up: ## Start Temporal server with Docker
	@echo "Starting Temporal server..."
	docker-compose up -d
	@echo "Temporal UI available at: http://localhost:8080"
	@echo "Waiting for Temporal to be ready..."
	@sleep 10

temporal-down: ## Stop Temporal server
	@echo "Stopping Temporal server..."
	docker-compose down

temporal-logs: ## Show Temporal server logs
	docker-compose logs -f temporal

worker: ## Run the Temporal worker
	@echo "Starting Temporal worker..."
	go run cmd/worker/main.go

web: ## Run the web server
	@echo "Starting web server..."
	go run cmd/web/main.go

dev: temporal-up ## Start development environment (Temporal + worker + web)
	@echo "Starting development environment..."
	@echo "Temporal UI: http://localhost:8080"
	@echo "Web App: http://localhost:8081"
	@echo ""
	@echo "Starting worker in background..."
	go run cmd/worker/main.go &
	@echo "Starting web server..."
	go run cmd/web/main.go

test: ## Run tests
	go test ./...

clean: ## Clean up containers and volumes
	docker-compose down -v
	docker system prune -f

build-worker: ## Build worker binary
	@echo "Building worker..."
	go build -o bin/worker cmd/worker/main.go

build-web: ## Build web binary
	@echo "Building web server..."
	go build -o bin/web cmd/web/main.go

build: build-worker build-web ## Build all binaries

install-temporal-cli: ## Install Temporal CLI (Mac)
	curl -sSf https://temporal.download/cli.sh | sh

workflow-list: ## List running workflows
	temporal workflow list --address localhost:7233

workflow-show: ## Show workflow details (usage: make workflow-show ID=workflow-id)
	temporal workflow show --workflow-id $(ID) --address localhost:7233

# Development shortcuts
start: temporal-up ## Alias for temporal-up
stop: temporal-down ## Alias for temporal-down
logs: temporal-logs ## Alias for temporal-logs