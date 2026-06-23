# Repository Guidelines

## Project Structure & Module Organization

This repository is a multi-service Travel AI application. The Next.js web app lives in `apps/web`, with routes in `src/app`, reusable UI in `src/components`, API clients in `src/lib/api`, and shared TypeScript types in `src/types`. Go services live under `services/*-service`; each follows the usual layout of `cmd/server`, `internal`, `pkg`, `configs`, and, where needed, `migrations`. The Python/FastAPI AI service is in `services/ai-planning-service`, with source in `app` and tests in `tests`. Local infrastructure is in `infra`, helper scripts are in `scripts`, and cross-cutting shared areas are reserved under `packages`.

## Build, Test, and Development Commands

- `cp infra/.env.example infra/.env` prepares local configuration.
- `./scripts/dev-setup.sh` performs local setup, including model preparation where applicable.
- `docker compose -f infra/docker-compose.yml --env-file infra/.env up --build` starts the full local stack at `http://localhost:3000`.
- `./scripts/smoke-test.sh` runs the full-stack API smoke test.
- From any Go service, run `make fmt`, `make vet`, `make test`, and `make build`.
- From `services/ai-planning-service`, run `make install`, `make lint`, `make fmt-check`, and `make test`.
- From `apps/web`, run `npm run dev`, `npm run build`, and `npm run typecheck`.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep package names short, lowercase, and domain-oriented. Use `internal` for service-private code and `pkg` only for code intended to be reused. Python targets 3.12, uses Ruff with 100-character lines, double quotes, and space indentation. TypeScript components use PascalCase filenames, hooks/helpers use camelCase, and route folders follow Next.js App Router conventions.

## Testing Guidelines

Go tests live beside implementation files as `*_test.go` and should be run with `make test`, which enables race detection. Python tests live in `services/ai-planning-service/tests` and use `test_*.py` naming with pytest. The web app currently exposes type checking and production build validation; add focused component or integration tests when introducing a test runner. Use `./scripts/smoke-test.sh` for changes touching service integration, authentication, trips, profiles, places, or itinerary generation.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit-style messages such as `feat: add itinerary version history`; keep subjects imperative and scoped to one change. Pull requests should include a brief summary, validation commands run, linked issue or context, screenshots for UI changes, and notes for migrations, environment variables, or service contract changes.

## Security & Configuration Tips

Do not commit real secrets. Start from `.env.example` files and keep local overrides out of git. Avoid logging access tokens, refresh tokens, or full user preference payloads.

## graphify

This project has a graphify knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `python3 -c "from graphify.watch import _rebuild_code; from pathlib import Path; _rebuild_code(Path('.'))"` to keep the graph current
