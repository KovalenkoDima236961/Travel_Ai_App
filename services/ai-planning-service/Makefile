.DEFAULT_GOAL := help

SERVICE     := ai-planning-service
HOST        ?= 0.0.0.0
PORT        ?= 8000
IMAGE       ?= $(SERVICE):local
FORMAT_PATHS := app tests docs/conf.py

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install local development dependencies
	python3 -m pip install -r requirements-dev.txt

.PHONY: poetry-install
poetry-install: ## Install local development dependencies with Poetry
	poetry install

.PHONY: poetry-lock
poetry-lock: ## Resolve Poetry dependencies
	poetry lock

.PHONY: run
run: ## Run the service locally
	python3 -m uvicorn app.main:app --host $(HOST) --port $(PORT) --reload

.PHONY: fmt
fmt: ## Format the codebase
	python3 -m isort $(FORMAT_PATHS)
	python3 -m black $(FORMAT_PATHS)
	python3 -m ruff format $(FORMAT_PATHS)

.PHONY: fmt-check
fmt-check: ## Check code formatting
	python3 -m isort --check-only --diff $(FORMAT_PATHS)
	python3 -m black --check $(FORMAT_PATHS)
	python3 -m ruff format --check $(FORMAT_PATHS)

.PHONY: lint
lint: ## Run lint checks
	python3 -m ruff check app tests
	python3 -m flake8 app tests

.PHONY: lint-fix
lint-fix: ## Run lint checks and apply safe fixes
	python3 -m ruff check app tests --fix

.PHONY: typecheck
typecheck: ## Run static type checks
	python3 -m mypy

.PHONY: test
test: ## Run tests
	python3 -m pytest

.PHONY: docs
docs: ## Build Sphinx HTML documentation
	python3 -m sphinx -W --keep-going -b html docs docs/_build/html

.PHONY: docs-clean
docs-clean: ## Remove generated Sphinx documentation
	rm -rf docs/_build

.PHONY: check
check: fmt-check lint typecheck test docs ## Run formatting, linting, type checks, tests, and docs

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(IMAGE) .

.PHONY: docker-run
docker-run: ## Run the Docker image
	docker run --rm -p $(PORT):8000 $(IMAGE)
