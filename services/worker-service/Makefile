.DEFAULT_GOAL := help

BINARY := worker-service
CMD    := ./cmd/worker

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the worker service
	go run $(CMD)

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

.PHONY: test
test: ## Run tests
	go test ./... -race -count=1
