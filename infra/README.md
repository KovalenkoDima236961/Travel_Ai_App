# Local Infrastructure

This folder contains the local Docker Compose stack for the full application.
The main compose file is `infra/docker-compose.yml`.

## Prerequisites

- Docker
- Docker Compose v2
- `jq` for `scripts/smoke-test.sh`

## Environment

Copy the example environment file before starting the stack:

```bash
cp infra/.env.example infra/.env
```

The compose file maps service-specific settings into the environment variable
names each service actually reads. `TRIP_ITINERARY_GENERATOR_MODE` and
`AI_ITINERARY_GENERATOR_MODE` are separate because both services read
`ITINERARY_GENERATOR_MODE` but need different values in the full local stack.

## Start The Stack

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

The web app is included in this stack and waits for Trip Service to become
healthy before starting. Auth Service also runs in the stack at
`http://localhost:8082`. Auth Service and Trip Service share
`JWT_ACCESS_SECRET` locally so Trip Service can validate Auth Service access
tokens without calling Auth Service on every request.

The helper scripts pass `--env-file infra/.env` explicitly. If you intentionally
use the shorter command below, confirm Docker Compose is picking up the right
environment values:

```bash
docker compose -f infra/docker-compose.yml up --build
```

Trip Service applies PostgreSQL migrations automatically on startup, so there is
no separate migration container in this stack. Auth Service does the same for
its own `auth_service` database. The Postgres init script in
`infra/postgres/init` creates `auth_service` when the database volume is first
initialized.

## Running The Web App

Start the full stack:

```bash
docker compose -f infra/docker-compose.yml up --build
```

Useful local URLs:

- Web App: http://localhost:3000
- Trip Service: http://localhost:8080
- Auth Service: http://localhost:8082
- AI Planning Service: http://localhost:8000

The `web-app` service receives both Trip Service URLs:

- `NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080` for browser-facing
  configuration.
- `NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8082` for browser-facing Auth
  Service calls.
- `TRIP_SERVICE_INTERNAL_URL=http://trip-service:8080` for server-side Next.js
  proxy calls inside Docker Compose.

Auth Service is exposed directly at `http://localhost:8082` for API testing and
the web app login/register flow.

Trip Service requires `Authorization: Bearer <accessToken>` on `/trips` routes
by default. To temporarily disable that for local debugging, set
`AUTH_REQUIRED=false` in `infra/.env`; unauthenticated trips will use
`DEV_USER_ID`.

Run the API smoke test from the repository root:

```bash
./scripts/smoke-test.sh
```

Run the manual browser flow in [scripts/web-smoke-test.md](../scripts/web-smoke-test.md).

## Pull Ollama Models

Run these once after Ollama is up:

```bash
docker compose -f infra/docker-compose.yml exec ollama ollama pull llama3.1:8b
docker compose -f infra/docker-compose.yml exec ollama ollama pull nomic-embed-text
```

Or use the setup helper:

```bash
./scripts/dev-setup.sh
```

## Index Knowledge

When `RAG_ENABLED=true`, index local knowledge files into the persisted ChromaDB
volume:

```bash
./scripts/index-knowledge.sh
```

This runs:

```bash
docker compose -f infra/docker-compose.yml run --rm ai-planning-service python -m app.scripts.index_knowledge
```

## Smoke Test

With the stack running:

```bash
./scripts/smoke-test.sh
```

The smoke test checks Auth Service, Trip Service, and AI Planning Service health,
registers and logs in a unique test user, calls `/auth/me`, creates a Rome trip
with a bearer token, generates its itinerary through the AI Planning Service,
fetches and lists the trip with the same token, verifies `status=COMPLETED` with
at least one itinerary day, registers a second user, confirms the second user
gets `404` for the first user's trip, then logs both users out.

The trip-service timeout must be longer than the AI service's Ollama timeout so
`OLLAMA_FALLBACK_TO_MOCK=true` has time to return a fallback itinerary. The
local defaults set `OLLAMA_TIMEOUT_SECONDS=90`,
`AI_PLANNING_TIMEOUT_SECONDS=120`, and `TRIP_HTTP_WRITE_TIMEOUT=150s`.

URLs can be overridden:

```bash
TRIP_SERVICE_URL=http://localhost:8080 \
AUTH_SERVICE_URL=http://localhost:8082 \
AI_PLANNING_SERVICE_URL=http://localhost:8000 \
WEB_APP_URL=http://localhost:3000 \
./scripts/smoke-test.sh
```

## Useful URLs

- Web App: http://localhost:3000
- Trip Service: http://localhost:8080
- Auth Service: http://localhost:8082
- AI Planning Service: http://localhost:8000
- Ollama: http://localhost:11434
- Adminer: http://localhost:8081

Adminer local defaults:

- System: PostgreSQL
- Server: `postgres`
- Username: `postgres`
- Password: `postgres`
- Database: `trip_service`

Use database `auth_service` in Adminer to inspect Auth Service users and
refresh tokens.

## Troubleshooting

- Browser CORS error: confirm `CORS_ALLOWED_ORIGINS=http://localhost:3000` is
  present in `infra/.env`, then rebuild/restart `trip-service` and
  `auth-service`. Both services only set `Access-Control-Allow-Origin` for
  configured origins.
- Trip request returns 401: login through the web app or call Auth Service
  directly, then send `Authorization: Bearer <accessToken>`. Confirm
  `JWT_ACCESS_SECRET` matches for both Auth Service and Trip Service.
- Web app cannot reach Trip Service from Docker: confirm
  `TRIP_SERVICE_INTERNAL_URL=http://trip-service:8080` is set for `web-app`.
- Browser points at the wrong Auth Service URL: confirm
  `NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8082` and rebuild the web app.
- Browser points at the wrong Trip Service URL: confirm
  `NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080` and rebuild the web app.
- Ollama model not found: run the two `ollama pull` commands above or rerun
  `./scripts/dev-setup.sh`.
- Ollama slow first response: the first local generation can take a while after
  model pull or container restart. Increase `OLLAMA_TIMEOUT_SECONDS` if needed.
  Keep `AI_PLANNING_TIMEOUT_SECONDS` higher than `OLLAMA_TIMEOUT_SECONDS`, and
  keep `TRIP_HTTP_WRITE_TIMEOUT` higher than `AI_PLANNING_TIMEOUT_SECONDS`.
- PostgreSQL not ready: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs postgres`.
- Migrations not run: Trip Service runs migrations during startup. Check
  `docker compose -f infra/docker-compose.yml logs trip-service` for migration
  or database connection errors.
- Auth database missing: the init script only runs when the Postgres volume is
  first created. For an existing local volume, create `auth_service` manually in
  Adminer or recreate the stack with `docker compose -f infra/docker-compose.yml
  down -v`.
- AI service falls back to mock: this is expected when
  `OLLAMA_FALLBACK_TO_MOCK=true` and Ollama generation fails. Check
  `docker compose -f infra/docker-compose.yml logs ai-planning-service`.
- ChromaDB/PostHog telemetry errors: local Chroma telemetry is disabled by
  default with `ANONYMIZED_TELEMETRY=false`. Rebuild/recreate
  `ai-planning-service` if older containers still log telemetry startup errors.
- RAG returns no results: run `./scripts/index-knowledge.sh`, confirm
  `RAG_ENABLED=true`, and verify the embedding model is pulled.
- ChromaDB data reset: Chroma persists in the `chroma-data` Docker volume. It is
  removed by `docker compose -f infra/docker-compose.yml down -v`.
