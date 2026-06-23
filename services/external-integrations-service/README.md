# External Integrations Service

External Integrations Service owns third-party integration boundaries for the travel app. v1 exposes place search and place details through a deterministic mock provider so the Web App can attach real-place shaped metadata without calling third-party APIs directly.

## Tech Stack

- Go
- `net/http` + chi router
- Uber Zap
- cleanenv config
- Docker
- No database in v1

The service uses the same explicit composition-root pattern as Auth Service,
Trip Service, and User Service. There is no DI framework in this service.

## Project Layout

```text
external-integrations-service/
├── cmd/server/main.go
├── configs/config.example.yaml
├── internal/
│   ├── app/                         # composition root
│   ├── application/service/         # use cases and provider port
│   ├── config/                      # cleanenv config
│   ├── domain/entity/               # Place entity
│   ├── http-server/                 # chi router, middleware, server
│   │   └── handler/                 # HTTP handlers and DTOs
│   └── infrastructure/provider/     # mock place provider adapter
├── pkg/closer
├── pkg/logger
├── Dockerfile
├── Makefile
└── docker-compose.yml
```

## Endpoints

- `GET /health`
- `GET /ready`
- `GET /places/search?query=Colosseum&destination=Rome`
- `GET /places/{placeId}`

Example:

```bash
curl "http://localhost:8084/places/search?query=Colosseum&destination=Rome"
```

## Configuration

- `APP_ENV` defaults to `development`
- `HTTP_ADDR` defaults to `:8084`
- `PLACE_PROVIDER` defaults to `mock`
- `CORS_ALLOWED_ORIGINS` defaults to `http://localhost:3000`
- `CORS_ALLOWED_METHODS` defaults to `GET,OPTIONS`
- `CORS_ALLOWED_HEADERS` defaults to `Content-Type,Authorization`

Documented for future providers, but unused in v1:

- `GOOGLE_PLACES_API_KEY`
- `MAPBOX_API_KEY`

Unsupported `PLACE_PROVIDER` values fail startup with a clear error.

Run locally with environment config:

```bash
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/server
```

Or with YAML config:

```bash
cp configs/config.example.yaml configs/config.yaml
go run ./cmd/server -config ./configs/config.yaml
```

## Places

The canonical place shape includes provider metadata, address, optional coordinates, rating, category, website, and a map URL. Place metadata is optional on itinerary items and is persisted by Trip Service as part of itinerary JSONB.

The mock provider includes deterministic places for Rome, Paris, Vienna, and Bratislava. Search is case-insensitive across place name, category, and address. When a destination is provided, results are filtered to that city. Unknown city-specific queries return a small fallback set for that city.

## Limitations

- Mock data only.
- No Google Places, Mapbox, or Foursquare provider is enabled yet.
- No full map view.
- No opening hours.
- No route optimization.
