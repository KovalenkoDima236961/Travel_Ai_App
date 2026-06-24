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
`http://localhost:8082`, User Service runs at `http://localhost:8083`, and
External Integrations Service runs at `http://localhost:8084`.
Auth Service, Trip Service, and User Service share `JWT_ACCESS_SECRET` locally
so downstream services can validate Auth Service access tokens without calling
Auth Service on every request.
During itinerary generation, Trip Service also calls User Service with the
current user's bearer token to load profile/preferences, then forwards optional
personalization context to AI Planning Service. `USER_CONTEXT_ENABLED=true`
enables this path and `USER_CONTEXT_FAIL_OPEN=true` lets generation continue
without personalization if User Service is unavailable.
Trip Service can also call External Integrations Service for mock weather
forecasts before full or partial itinerary generation. `WEATHER_CONTEXT_ENABLED=true`
enables this path and `WEATHER_CONTEXT_FAIL_OPEN=true` lets generation continue
without weather if the service is unavailable.
Trip Service also uses External Integrations Service for AI Place Enrichment v1
after generated itinerary payloads. `PLACE_ENRICHMENT_ENABLED=true` enables the
path, `PLACE_ENRICHMENT_FAIL_OPEN=true` keeps generation successful when place
search is unavailable, and `PLACE_ENRICHMENT_MIN_CONFIDENCE=0.75`,
`PLACE_ENRICHMENT_MAX_ITEMS=20`, and
`PLACE_ENRICHMENT_OVERWRITE_EXISTING=false` control matching behavior.
Public Trip Sharing v1 is enabled by default with
`PUBLIC_SHARING_ENABLED=true`, builds owner-facing links from
`PUBLIC_WEB_BASE_URL=http://localhost:3000`, and creates opaque share tokens
from `SHARE_TOKEN_BYTES=32` random bytes. Share Controls v1 adds optional link
expiration and password protection. Protected public links unlock through a
short-lived public share token signed with
`PUBLIC_SHARE_ACCESS_SECRET=dev-public-share-secret-change-me` for local
development and expiring after `PUBLIC_SHARE_ACCESS_TTL_MINUTES=60`. This token
is separate from Auth Service JWTs and is scoped to one share token.
Collaborative Planning v1 is enabled by Trip Service's
`AUTH_SERVICE_URL=http://auth-service:8082` internal lookup. Owners invite
existing registered users by exact email; Auth Service resolves the email to a
user ID. The internal lookup endpoint is network-internal only in this Compose
stack and is not service-authenticated yet.

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
initialized. Trip Service migrations create `trips`, `itinerary_versions`, and
`trip_shares`; version history starts from the point the migration is deployed
and existing itineraries are not backfilled.

## Running The Web App

Start the full stack:

```bash
docker compose -f infra/docker-compose.yml up --build
```

Useful local URLs:

- Web App: http://localhost:3000
- Web Settings Page: http://localhost:3000/settings
- Trip Service: http://localhost:8080
- Auth Service: http://localhost:8082
- User Service: http://localhost:8083
- External Integrations Service: http://localhost:8084
- AI Planning Service: http://localhost:8000

The `web-app` service receives browser-facing and internal service URLs:

- `NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080` for browser-facing
  configuration.
- `NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8082` for browser-facing Auth
  Service calls.
- `NEXT_PUBLIC_USER_SERVICE_URL=http://localhost:8083` for browser-facing User
  Service calls.
- `NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL=http://localhost:8084` for
  browser-facing External Integrations Service calls.
- `TRIP_SERVICE_INTERNAL_URL=http://trip-service:8080` for server-side Next.js
  proxy calls inside Docker Compose.
- `USER_SERVICE_INTERNAL_URL=http://user-service:8083` for future server-side
  Next.js proxy calls inside Docker Compose.

Trip Service also receives:

- `AUTH_SERVICE_URL=http://auth-service:8082` inside the container, sourced from
  `AUTH_SERVICE_INTERNAL_URL`, for exact-email collaborator invite lookup.
- `PUBLIC_WEB_BASE_URL=http://localhost:3000` for share URLs returned to owners.
- `PUBLIC_SHARING_ENABLED=true` to enable owner-managed public links.
- `SHARE_TOKEN_BYTES=32` to keep share tokens cryptographically random and
  base64url encoded.
- `PUBLIC_SHARE_ACCESS_SECRET=dev-public-share-secret-change-me` for local
  password-protected share unlock tokens. Change it outside development and keep
  it different from `JWT_ACCESS_SECRET`.
- `PUBLIC_SHARE_ACCESS_TTL_MINUTES=60` for public share unlock token lifetime.

Auth Service is exposed directly at `http://localhost:8082` for API testing and
the web app login/register flow.

User Service is exposed directly at `http://localhost:8083` for profile and
preferences API testing. It owns travel preferences; Auth Service owns
identity, Trip Service owns trips, and AI Planning Service owns itinerary
generation.

External Integrations Service is exposed directly at `http://localhost:8084`.
The local default uses `PLACE_PROVIDER=mock`, `ROUTE_PROVIDER=mock`, and
`WEATHER_PROVIDER=mock`, returning deterministic data for local development.
Mock place search/details can include optional `openingHours` intervals using
`dayOfWeek` `1 = Monday` through `7 = Sunday` and local `HH:mm` times. These
hours are advisory only: there is no timezone, holiday, special-date, or real
opening-hours provider in v1.
Examples:

```bash
curl "http://localhost:8084/places/search?query=Colosseum&destination=Rome"

curl -X POST "http://localhost:8084/routes/estimate" \
  -H "Content-Type: application/json" \
  -d '{"mode":"walking","stops":[
    {"name":"Colosseum","latitude":41.8902,"longitude":12.4922},
    {"name":"Trevi Fountain","latitude":41.9009,"longitude":12.4833}
  ]}'

curl "http://localhost:8084/weather/forecast?destination=Rome&startDate=2026-08-10&days=3"
```

`POST /routes/estimate` returns approximate mock walking route estimates
(Haversine × 1.25 at 5 km/h). Because the Web App calls it from the browser, the
service's CORS allows `POST` (`CORS_ALLOWED_METHODS=GET,POST,OPTIONS` by
default). Weather forecast calls are read-only `GET` requests. Future provider
candidates include Google Places, Mapbox, OSRM, Google Maps routing,
Open-Meteo, and real opening-hours providers.

### Place Provider

Mock provider, no API key required:

```bash
PLACE_PROVIDER=mock
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

Foursquare provider:

1. Set `PLACE_PROVIDER=foursquare` in `infra/.env`.
2. Set `FOURSQUARE_API_KEY=...`.
3. Optionally keep `PLACE_PROVIDER_FALLBACK_TO_MOCK=true` for local fallback.
4. Start the stack:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

Troubleshooting:

- Missing API key: with fallback enabled, development falls back to mock and
  logs `fallbackUsed=true`; with fallback disabled, startup fails.
- `401` or `403`: the Foursquare API key is missing, invalid, or unauthorized.
- `429`: Foursquare rate limit reached.
- Provider `5xx` or malformed responses: the service returns a stable app error
  unless fallback is enabled, in which case it logs the fallback and returns mock
  results.
- Real provider results may omit rating, website, coordinates, or opening hours;
  those fields are optional in the Web App.

Trip Service requires `Authorization: Bearer <accessToken>` on `/trips` routes
by default. To temporarily disable that for local debugging, set
`AUTH_REQUIRED=false` in `infra/.env`; unauthenticated trips will use
`DEV_USER_ID`.

Run the API smoke test from the repository root:

```bash
./scripts/smoke-test.sh
```

Run the manual browser flow in [scripts/web-smoke-test.md](../scripts/web-smoke-test.md).

Manual personalization check:

1. Register or log in.
2. Open http://localhost:3000/settings.
3. Update profile and travel preferences.
4. Create a trip.
5. Generate the itinerary.
6. Confirm the itinerary reflects the saved preferences.

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

The smoke test checks Auth Service, Trip Service, User Service, External
Integrations Service, and AI Planning Service health, registers and logs in a
unique test user, calls `/auth/me`, creates/updates profile and preferences with
a bearer token, searches mock places for Colosseum in Rome, requests a mock
walking route estimate and asserts a `mock` provider with positive distance,
duration, and one segment (and that fewer than two stops is rejected), requests a
mock Rome weather forecast and checks provider/day data, creates a Rome trip,
generates its itinerary through Trip Service's personalized context path, creates
and disables a public share link while confirming `/public/trips/{shareToken}`
works without Authorization and does not expose private fields, saves a manual
itinerary edit with attached place metadata and opening hours, verifies
the metadata persists after fetch and in the manual-edit version snapshot, lists
itinerary versions, checks `GENERATED`, `MANUAL_EDIT`, and `RESTORED` version
sources, restores the generated version, warns if avoided nightlife wording
appears, registers a second user, confirms the second user gets `404` for the
first user's trip and itinerary versions, then logs both users out.

The trip-service timeout must be longer than the AI service's Ollama timeout so
`OLLAMA_FALLBACK_TO_MOCK=true` has time to return a fallback itinerary. The
local defaults set `OLLAMA_TIMEOUT_SECONDS=90`,
`AI_PLANNING_TIMEOUT_SECONDS=120`, and `TRIP_HTTP_WRITE_TIMEOUT=150s`.

URLs can be overridden:

```bash
TRIP_SERVICE_URL=http://localhost:8080 \
AUTH_SERVICE_URL=http://localhost:8082 \
SMOKE_USER_SERVICE_URL=http://localhost:8083 \
SMOKE_AI_PLANNING_SERVICE_URL=http://localhost:8000 \
SMOKE_EXTERNAL_INTEGRATIONS_SERVICE_URL=http://localhost:8084 \
WEB_APP_URL=http://localhost:3000 \
./scripts/smoke-test.sh
```

The smoke script also tolerates `USER_SERVICE_URL=http://user-service:8083` and
`AI_PLANNING_SERVICE_URL=http://ai-planning-service:8000` from a sourced
`infra/.env` by mapping those internal Docker hostnames back to localhost.

## Useful URLs

- Web App: http://localhost:3000
- Web Settings Page: http://localhost:3000/settings
- Trip Service: http://localhost:8080
- Auth Service: http://localhost:8082
- User Service: http://localhost:8083
- External Integrations Service: http://localhost:8084
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
refresh tokens. Use database `user_service` to inspect profiles and preferences.
Use database `trip_service` to inspect trips and itinerary version snapshots.

## Troubleshooting

- Browser CORS error: confirm `CORS_ALLOWED_ORIGINS=http://localhost:3000` is
  present in `infra/.env`, then rebuild/restart `trip-service` and
  `auth-service`. Both services only set `Access-Control-Allow-Origin` for
  configured origins.
- Trip request returns 401: login through the web app or call Auth Service
  directly, then send `Authorization: Bearer <accessToken>`. Confirm
  `JWT_ACCESS_SECRET` matches for Auth Service, Trip Service, and User Service.
- User profile/preferences request returns 401: confirm the same bearer token
  and shared `JWT_ACCESS_SECRET` are used.
- Personalized generation returns `failed to load user preferences`: User
  Service is unavailable and `USER_CONTEXT_FAIL_OPEN=false`. Restore User
  Service or set `USER_CONTEXT_FAIL_OPEN=true` for local fail-open behavior.
- Generation returns `failed to load weather forecast`: External Integrations
  Service is unavailable and `WEATHER_CONTEXT_FAIL_OPEN=false`. Restore the
  service or set `WEATHER_CONTEXT_FAIL_OPEN=true` for local fail-open behavior.
- Web app cannot reach Trip Service from Docker: confirm
  `TRIP_SERVICE_INTERNAL_URL=http://trip-service:8080` is set for `web-app`.
- Browser points at the wrong Auth Service URL: confirm
  `NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8082` and rebuild the web app.
- Browser points at the wrong Trip Service URL: confirm
  `NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080` and rebuild the web app.
- Place search fails in the browser: confirm
  `NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL=http://localhost:8084`, confirm
  External Integrations Service is healthy, and confirm
  `CORS_ALLOWED_ORIGINS=http://localhost:3000`.
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
- Auth or user database missing: the init script only runs when the Postgres
  volume is first created. For an existing local volume, create `auth_service`
  or `user_service` manually in Adminer, or recreate the stack with
  `docker compose -f infra/docker-compose.yml down -v`.
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
