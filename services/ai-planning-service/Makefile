.DEFAULT_GOAL := help

SERVICE     := ai-planning-service
HOST        ?= 0.0.0.0
PORT        ?= 8000
IMAGE       ?= $(SERVICE):local

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install local development dependencies
	python3 -m pip install -r requirements-dev.txt

.PHONY: run
run: ## Run the service locally
	python3 -m uvicorn app.main:app --host $(HOST) --port $(PORT) --reload

.PHONY: fmt
fmt: ## Format the codebase
	python3 -m ruff format app tests

.PHONY: fmt-check
fmt-check: ## Check code formatting
	python3 -m ruff format --check app tests

.PHONY: lint
lint: ## Run lint checks
	python3 -m ruff check app tests

.PHONY: lint-fix
lint-fix: ## Run lint checks and apply safe fixes
	python3 -m ruff check app tests --fix

.PHONY: test
test: ## Run tests
	python3 -m pytest

.PHONY: check
check: fmt-check lint test ## Run formatting, lint, and tests

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(IMAGE) .

.PHONY: docker-run
docker-run: ## Run the Docker image
	docker run --rm -p $(PORT):8000 $(IMAGE)
