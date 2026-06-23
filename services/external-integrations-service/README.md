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
│   ├── domain/entity/               # Place and Route entities
│   ├── http-server/                 # chi router, middleware, server
│   │   └── handler/                 # HTTP handlers and DTOs
│   └── infrastructure/provider/     # mock place and route provider adapters
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
- `POST /routes/estimate`

Example:

```bash
curl "http://localhost:8084/places/search?query=Colosseum&destination=Rome"
```

## Configuration

- `APP_ENV` defaults to `development`
- `HTTP_ADDR` defaults to `:8084`
- `PLACE_PROVIDER` defaults to `mock`
- `ROUTE_PROVIDER` defaults to `mock`
- `CORS_ALLOWED_ORIGINS` defaults to `http://localhost:3000`
- `CORS_ALLOWED_METHODS` defaults to `GET,POST,OPTIONS`
- `CORS_ALLOWED_HEADERS` defaults to `Content-Type,Authorization`

Documented for future providers, but unused in v1:

- `GOOGLE_PLACES_API_KEY`
- `MAPBOX_API_KEY`
- `OSRM_BASE_URL`
- `MAPBOX_ACCESS_TOKEN`
- `GOOGLE_MAPS_API_KEY`

Unsupported `PLACE_PROVIDER` or `ROUTE_PROVIDER` values fail startup with a
clear error. The two providers are selected independently.

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

## Routing API v1

`POST /routes/estimate` returns an approximate travel-time/distance estimate for
an ordered list of stops. v1 ships only a deterministic `mock` provider behind a
`RouteProvider` port, so real providers (OSRM, Mapbox, Google) can be added later
without changing the API.

Request:

```bash
curl -X POST "http://localhost:8084/routes/estimate" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "walking",
    "stops": [
      {"name": "Colosseum", "latitude": 41.8902, "longitude": 12.4922},
      {"name": "Trevi Fountain", "latitude": 41.9009, "longitude": 12.4833}
    ]
  }'
```

Response:

```json
{
  "mode": "walking",
  "provider": "mock",
  "distanceKm": 2.1,
  "durationMinutes": 28,
  "segments": [
    {
      "fromName": "Colosseum",
      "toName": "Trevi Fountain",
      "distanceKm": 2.1,
      "durationMinutes": 28
    }
  ]
}
```

How the mock provider estimates each consecutive stop pair:

- **Supported mode:** `walking` only.
- **Provider:** `mock`.
- **Distance:** Haversine straight-line distance × a `1.25` route factor, rounded
  to 2 decimals.
- **Walking speed:** flat `5 km/h`; `durationMinutes = round(distanceKm / 5 * 60)`.
- **Totals:** `distanceKm` and `durationMinutes` are the sum of the segment
  values, so the total always equals the sum of the segments the caller sees.

Validation (returns `400` with `{"error": "..."}`):

- `mode` is required and must be `walking`.
- `stops` is required, with a minimum of 2 and a maximum of 25 stops.
- each stop requires a `name` (≤ 200 chars) and valid coordinates
  (`latitude` ∈ [-90, 90], `longitude` ∈ [-180, 180]).

Limitations:

- Not real routing — straight-line distance scaled by a constant factor.
- No traffic, no elevation, no turn-by-turn geometry, no route polyline.
- No public-transport, driving, or cycling modes yet.
- The Web App uses these estimates read-only and falls back to its own Haversine
  straight-line estimate when the service is unavailable.

## Limitations

- Mock data only.
- No Google Places, Mapbox, or Foursquare provider is enabled yet.
- No full map view.
- No opening hours.
- No real routing — `POST /routes/estimate` returns approximate mock walking
  estimates (Haversine × 1.25), not turn-by-turn directions.
