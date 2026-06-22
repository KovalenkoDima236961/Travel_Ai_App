# Trip Service

A Go microservice for an AI travel planning web app. It manages **trip requests**
(destination, dates, budget, travelers, interests, pace) and generates an
**itinerary** through a configurable generator: either a deterministic local mock
or AI Planning Service v1 over HTTP.

## Tech stack

| Concern           | Choice                                                        |
| ----------------- | ------------------------------------------------------------ |
| Language          | Go                                                           |
| Composition / DI  | Hand-wired composition root (`internal/app`) — no framework  |
| Lifecycle         | `pkg/closer` (LIFO graceful shutdown)                        |
| Logging           | [Uber Zap](https://github.com/uber-go/zap) (`pkg/logger`)   |
| HTTP              | `net/http` + [chi](https://github.com/go-chi/chi) router    |
| Database          | PostgreSQL via [pgx](https://github.com/jackc/pgx) (`pgxpool`) |
| Query building    | [squirrel](https://github.com/Masterminds/squirrel)         |
| Migrations        | [golang-migrate](https://github.com/golang-migrate/migrate) — **applied automatically on startup** |
| Config            | YAML + env via [cleanenv](https://github.com/ilyakaznacheev/cleanenv) |
| Validation        | [go-playground/validator](https://github.com/go-playground/validator) (`pkg/validation`) |
| Container         | Docker / Docker Compose                                     |

> This service lives at `services/trip-service/` within the monorepo. All commands
> below assume that directory as the working directory. The Go module path is
> `github.com/KovalenkoDima236961/Travel_Ai_App`.

## Project layout

The code follows a layered / hexagonal (DDD-flavoured) structure under `internal/`:

```
trip-service/
├── cmd/server/main.go                 # entrypoint: app.New(configPath).Run()
├── internal/
│   ├── app/                           # composition root (app.go) + wiring (di.go)
│   ├── config/                        # cleanenv config + validation
│   ├── domain/                        # enterprise core (no outward deps)
│   │   ├── entity/                    #   Trip, Status
│   │   ├── aggregate/                 #   Itinerary, ItineraryDay, ItineraryItem
│   │   └── errs/                      #   ErrNotFound (domain sentinel)
│   ├── application/                   # use cases + ports
│   │   ├── service/                   #   Service (business logic, tripRepository port)
│   │   ├── dto/                       #   CreateTripInput (use-case input)
│   │   ├── errs/                      #   InvalidInputError
│   │   └── generator.go               #   ItineraryGenerator (port interface)
│   ├── infrastructure/                # adapters (implement ports)
│   │   ├── repository/postgres/       #   Repository (squirrel) + dto/ (pgtype ⇄ entity)
│   │   └── generator/                 #   Mock + AI Planning HTTP generator adapters
│   └── http-server/                   # delivery: chi router + http.Server
│       ├── handler/                   #   Handler (decode/validate/status mapping)
│       └── dto/{request,response}/    #   CreateTrip / Trip + ListTrips payloads
├── pkg/
│   ├── closer/                        # global LIFO shutdown registry
│   ├── logger/                        # zap logger
│   ├── storage/postgres/              # pgxpool + squirrel builder + auto-migrate
│   ├── cache/redis/                   # redis client (available, not wired)
│   ├── tls/                           # autocert TLS manager (available, not wired)
│   └── validation/                    # validator wrapper + custom tags
├── configs/config.example.yaml
├── migrations/                        # golang-migrate up/down SQL
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

### Layering

Dependencies point inward: `http-server` → `application` → `domain`, with
`infrastructure` adapters implementing the application's ports. `domain` imports
nothing else in the project.

```
HTTP request → http-server/handler          (decode + validate + status mapping)
             → application/service           (defaults, business rules, transitions)
             → application ports:
                 • tripRepository  → infrastructure/repository/postgres (squirrel)
                 • ItineraryGenerator → infrastructure/generator (mock/http)
             → pkg/storage/postgres (pgxpool) → PostgreSQL
```

## Architecture (Mermaid)

```mermaid
flowchart TD
    Client["Web / Admin client"]

    subgraph TS["trip-service"]
        direction TB
        RT["http-server: chi router + middleware"]
        TH["trip.Handler\n(decode / validate / status)"]
        SV["trip.Service\n(defaults, business rules)"]
        GEN["trip.ItineraryGenerator\n(Mock or HTTP adapter)"]
        RP["trip.Repository\n(squirrel + pgtype ⇄ domain)"]
        PGPKG["pkg/storage/postgres\n(pgxpool + squirrel builder)"]
    end

    PG[("PostgreSQL\ntrips table")]
    AI{{"AI Planning Service v1\nFastAPI / HTTP"}}

    Client -->|"REST / JSON"| RT --> TH --> SV --> RP --> PGPKG -->|"pgxpool"| PG
    SV --> GEN
    GEN -.->|"POST /generate-itinerary\nwhen mode=http"| AI

    subgraph BOOT["internal/app (composition root)"]
        CFG["config (cleanenv)"] --> LOG["zap logger"] --> DB["postgres.New\n(auto-migrate)"] --> SRV["http server"]
        CL["pkg/closer (LIFO)"]
    end
```

`internal/app` is a small, explicit composition root (no DI framework). On startup
it loads + validates config, builds the logger, opens the pool (running migrations
automatically), wires the trip feature, and starts the HTTP server. Long-lived
resources register with `pkg/closer`; on `SIGINT`/`SIGTERM` they are closed LIFO
(HTTP server drained first, then the DB pool).

## Configuration

Config is read from a YAML file (via the `-config` flag) **and/or** environment
variables (env overrides file). When no `-config` is passed, it is loaded from the
environment only. It is then validated with `pkg/validation`.

Use [.env.example](.env.example) as the local env template:

```bash
cp .env.example .env
set -a; source .env; set +a
```

Key environment variables:

| Variable             | Default        | Description                          |
| -------------------- | -------------- | ------------------------------------ |
| `APP_ENV`            | `development`  | `development` or `production`.       |
| `HTTP_ADDRESS`       | `:8080`        | HTTP listen address.                 |
| `POSTGRES_HOST`      | —              | Database host.                       |
| `POSTGRES_PORT`      | —              | Database port.                       |
| `POSTGRES_DB`        | —              | Database name.                       |
| `POSTGRES_USER`      | —              | Database user.                       |
| `POSTGRES_PASSWORD`  | —              | Database password.                   |
| `POSTGRES_MIN_CONNS` | —              | Pool minimum connections (≥ 1).      |
| `POSTGRES_MAX_CONNS` | —              | Pool maximum connections (≥ 1).      |
| `POSTGRES_MIG_PATH`  | —              | Path to the `migrations/` directory. |
| `ITINERARY_GENERATOR_MODE` | `mock` | `mock` for local generation, `http` for AI Planning Service. |
| `AI_PLANNING_SERVICE_URL` | `http://ai-planning-service:8000` | Base URL used when generator mode is `http`. |
| `AI_PLANNING_TIMEOUT_SECONDS` | `10` | HTTP client timeout for AI Planning Service calls. |

See [configs/config.example.yaml](configs/config.example.yaml) for the file form.

Unknown generator modes fail startup. In `http` mode, startup also fails if
`AI_PLANNING_SERVICE_URL` is missing or invalid.

## Run with Docker Compose

```bash
docker compose up --build
```

Brings up PostgreSQL, AI Planning Service, and Trip Service (configured via env).
Trip Service runs with `ITINERARY_GENERATOR_MODE=http` and calls
`http://ai-planning-service:8000/generate-itinerary`. The service applies
migrations itself on startup, so there is no separate migrate step. The Trip API
is available at `http://localhost:8080`; AI Planning Service is exposed at
`http://localhost:8000`.

Tear down (and wipe the DB volume):

```bash
docker compose down -v
```

## Run locally (without Docker)

```bash
# 1. Start Postgres
docker run --rm -d --name trip-pg \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=trip_service \
  -p 5432:5432 postgres:16-alpine

# 2a. Run with env config and the local mock generator
export APP_ENV=development HTTP_ADDRESS=":8080" \
  POSTGRES_DB=trip_service POSTGRES_USER=postgres POSTGRES_PASSWORD=postgres \
  POSTGRES_HOST=localhost POSTGRES_PORT=5432 \
  POSTGRES_MIN_CONNS=2 POSTGRES_MAX_CONNS=10 POSTGRES_MIG_PATH=./migrations \
  ITINERARY_GENERATOR_MODE=mock
go run ./cmd/server

# 2b. Or keep the database env vars above and switch to AI Planning Service over HTTP.
# In another shell from services/ai-planning-service:
#   uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
export ITINERARY_GENERATOR_MODE=http \
  AI_PLANNING_SERVICE_URL=http://localhost:8000 \
  AI_PLANNING_TIMEOUT_SECONDS=10
go run ./cmd/server

# 2c. …or run with a YAML config file
cp configs/config.example.yaml configs/config.yaml
go run ./cmd/server -config ./configs/config.yaml
```

Common tasks are also available via the Makefile (`make help`).

## Migrations

Migrations are applied **automatically** on startup by `pkg/storage/postgres`
(golang-migrate). To apply them manually instead (e.g. in CI) with the
[migrate](https://github.com/golang-migrate/migrate) CLI:

```bash
make migrate-up      # or: migrate -path ./migrations -database "$DB_URL" up
make migrate-down    # roll back the last migration
```

## API

| Method | Path                    | Description                                  |
| ------ | ----------------------- | -------------------------------------------- |
| GET    | `/health`               | Liveness probe.                              |
| POST   | `/trips`                | Create a trip (status `DRAFT`).              |
| GET    | `/trips`                | List trips (paginated, newest first).        |
| GET    | `/trips/{id}`           | Fetch a trip by UUID.                        |
| POST   | `/trips/{id}/generate`  | Generate the itinerary with the configured generator; status `COMPLETED`. |

Trip statuses: `DRAFT` → `PROCESSING` → `COMPLETED` (or `FAILED`).

### Itinerary generation

Itinerary generation is abstracted behind a `trip.ItineraryGenerator` interface, so
the service layer does not depend on any particular planning strategy. The
configured adapter is selected at startup:

| Mode | Behavior |
| ---- | -------- |
| `mock` | Uses `MockItineraryGenerator` locally. This is the default when mode is empty. |
| `http` | Uses `AIPlanningHTTPGenerator` to call `POST {AI_PLANNING_SERVICE_URL}/generate-itinerary`. |

The HTTP adapter sends the Trip fields as JSON, uses the configured client
timeout, decodes the AI Planning Service response into typed `Itinerary` /
`ItineraryDay` / `ItineraryItem` structs, and returns an error for non-2xx,
invalid JSON, request failures, or empty `days`. Generated plans are stored as
JSONB by the service layer.

Errors use a uniform envelope; validation failures add a `fields` map:

```json
{ "error": "validation failed", "fields": { "Days": "'Days' must be <= 30" } }
```

### Example curl requests

```bash
TRIP_ID=$(curl -s -X POST http://localhost:8080/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 4,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history", "hidden_gems"],
    "pace": "balanced"
  }' | jq -r '.id')

# Generate the itinerary
curl -s -X POST "http://localhost:8080/trips/${TRIP_ID}/generate"

# Fetch the completed trip
curl -s "http://localhost:8080/trips/${TRIP_ID}"

# List (paginated, newest first)
curl -s "http://localhost:8080/trips?limit=20&offset=0"

# Health
curl -s http://localhost:8080/health
```

The list endpoint returns a paginated envelope:

```json
{
  "items": [ { "id": "…", "destination": "Rome", "status": "COMPLETED", "itinerary": { } } ],
  "limit": 20,
  "offset": 0
}
```

## Validation rules

- `destination` is required.
- `days` is required and must be between 1 and 30.
- `travelers` is required and must be ≥ 1.
- `budgetCurrency`, when present, must be 3 characters (defaults to `EUR` when empty).
- `pace`, when present, must be one of `relaxed | balanced | packed` (defaults to `balanced`).
- `startDate`, when present, must be `YYYY-MM-DD`.
- `interests`, when omitted, defaults to an empty array.

For `GET /trips`:

- `limit` defaults to `20`, and must be between `1` and `100`.
- `offset` defaults to `0`, and must be `>= 0`.

## Tests

Service-level business logic is covered by unit tests that mock the repository and
the itinerary generator (no database required):

```bash
make test          # go test ./... -race -count=1
# or directly:
go test ./... -race -count=1
```

## Notes & extension points

- The `id` column uses `gen_random_uuid()` (PostgreSQL 13+) so IDs are generated
  by the database.
- Itinerary generation is selected by `ITINERARY_GENERATOR_MODE`. Use `mock` for
  local deterministic output or `http` for AI Planning Service v1. RabbitMQ, real
  LLM calls, and RAG are intentionally not part of this version.
- `pkg/cache/redis` and `pkg/tls` are wired-ready platform utilities included for
  future use; the trip feature does not depend on them yet.
