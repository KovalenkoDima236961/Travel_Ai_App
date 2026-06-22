.DEFAULT_GOAL := help

BINARY      := trip-service
CMD         := ./cmd/server
CONFIG      ?= ./configs/config.yaml
MIGRATIONS  := ./migrations
DB_URL      ?= postgres://postgres:postgres@localhost:5432/trip_service?sslmode=disable

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the service (env-based config; set vars or use config-run)
	go run $(CMD)

.PHONY: config-run
config-run: ## Run the service with a YAML config file
	go run $(CMD) -config $(CONFIG)

.PHONY: build
build: ## Build the binary into ./bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY) $(CMD)

.PHONY: tidy
tidy: ## Sync go.mod/go.sum
	go mod tidy

.PHONY: fmt
fmt: ## Format the codebase
	gofmt -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (uses .golangci.yml)
	golangci-lint run ./...

.PHONY: test
test: ## Run tests
	go test ./... -race -count=1

.PHONY: migrate-up
migrate-up: ## Apply migrations manually (the app also auto-migrates on start)
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

.PHONY: docker-up
docker-up: ## Start the full stack (Postgres + service)
	docker compose up --build

.PHONY: docker-down
docker-down: ## Stop the stack and remove volumes
	docker compose down -v
