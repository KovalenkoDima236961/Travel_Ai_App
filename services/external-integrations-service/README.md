# External Integrations Service

External Integrations Service owns third-party integration boundaries for the
travel app. v1 exposes place search/details, route estimates, weather forecasts,
exchange-rate conversion, and attraction/ticket price estimates through stable
application APIs so the Web App and Trip Service can use integration-shaped data
without calling third-party APIs directly. Mock providers remain the local
default; place search/details can optionally use Foursquare.
Calendar Sync v1 also lives here: the service owns Google OAuth, encrypted token
storage, and Google Calendar event create/update/delete calls for Trip Service.

## Tech Stack

- Go
- `net/http` + chi router
- Uber Zap
- cleanenv config
- Docker
- PostgreSQL for calendar OAuth state and token storage

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
│   ├── domain/entity/               # Place, Route, and Weather entities
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
- `GET /weather/forecast?destination=Rome&startDate=2026-08-10&days=3`
- `GET /exchange-rates/latest?base=EUR`
- `GET /exchange-rates/convert?amount=2500&from=JPY&to=EUR`
- `POST /prices/estimate` (internal service-token route)
- `GET /calendar/google/status`
- `POST /calendar/google/connect`
- `GET /calendar/google/callback`
- `DELETE /calendar/google/disconnect`
- `POST /internal/calendar/google/events/sync`
- `POST /internal/calendar/google/events/delete`

Example:

```bash
curl "http://localhost:8084/places/search?query=Colosseum&destination=Rome"
```

## Configuration

- `APP_ENV` defaults to `development`
- `HTTP_ADDR` defaults to `:8084`
- `PLACE_PROVIDER` defaults to `mock`
- `PLACE_PROVIDER_FALLBACK_TO_MOCK` defaults to `true`
- `FOURSQUARE_API_KEY` is required when `PLACE_PROVIDER=foursquare` and fallback
  is disabled
- `FOURSQUARE_BASE_URL` defaults to `https://api.foursquare.com/v3`
- `FOURSQUARE_TIMEOUT_SECONDS` defaults to `8`
- `ROUTE_PROVIDER` defaults to `mock` (`mock` | `ors`)
- `ROUTE_PROVIDER_FALLBACK_TO_MOCK` defaults to `true`
- `ROUTE_PROVIDER_TIMEOUT_SECONDS` defaults to `8`
- `ORS_API_KEY` is required when `ROUTE_PROVIDER=ors` and fallback is disabled
- `ORS_BASE_URL` defaults to `https://api.openrouteservice.org`
- `ORS_PROFILE_WALKING` / `ORS_PROFILE_DRIVING` / `ORS_PROFILE_CYCLING` default to
  `foot-walking` / `driving-car` / `cycling-regular`
- `ROUTE_CACHE_ENABLED` defaults to `true`; `ROUTE_CACHE_TTL_SECONDS` defaults to
  `21600` (6 hours)
- `WEATHER_PROVIDER` defaults to `mock` (`mock` | `openweathermap`)
- `WEATHER_PROVIDER_FALLBACK_TO_MOCK` defaults to `true`
- `WEATHER_PROVIDER_TIMEOUT_SECONDS` defaults to `8`
- `OPENWEATHER_API_KEY` is required when `WEATHER_PROVIDER=openweathermap` and
  fallback is disabled
- `OPENWEATHER_BASE_URL` defaults to `https://api.openweathermap.org`
- `OPENWEATHER_UNITS` defaults to `metric`
- `WEATHER_CACHE_ENABLED` defaults to `true`; `WEATHER_CACHE_TTL_SECONDS` defaults
  to `3600` (1 hour)
- `EXCHANGE_RATE_PROVIDER` defaults to `mock`. Reserved real-provider names:
  `exchangerate_host`, `openexchangerates`, `exchangerate_api`.
- `EXCHANGE_RATE_PROVIDER_FALLBACK_TO_MOCK` defaults to `true`.
- `EXCHANGE_RATE_PROVIDER_TIMEOUT_SECONDS` defaults to `8`.
- `EXCHANGE_RATE_BASE_URL` and `EXCHANGE_RATE_API_KEY` are reserved for real
  providers. API keys are server-side only and must not be exposed to the Web App.
- `EXCHANGE_RATE_CACHE_ENABLED` defaults to `true`;
  `EXCHANGE_RATE_CACHE_TTL_SECONDS` defaults to `21600` (6 hours).
- `PRICE_PROVIDER` defaults to `mock`. Reserved real-provider name: `api`.
- `PRICE_PROVIDER_FALLBACK_TO_MOCK` defaults to `true`.
- `PRICE_PROVIDER_TIMEOUT_SECONDS` defaults to `8`.
- `PRICE_CACHE_ENABLED` defaults to `true`; `PRICE_CACHE_TTL_SECONDS` defaults to
  `86400` (24 hours).
- `PRICE_ENRICHMENT_DEFAULT_CURRENCY` supplies the default currency for price
  estimates and defaults to `EUR`.
- `PRICE_API_BASE_URL` and `PRICE_API_KEY` are reserved for real price providers.
  API keys are server-side only and must not be exposed to the Web App.
- `CORS_ALLOWED_ORIGINS` defaults to `http://localhost:3000`
- `CORS_ALLOWED_METHODS` defaults to `GET,POST,DELETE,OPTIONS`
- `CORS_ALLOWED_HEADERS` defaults to `Content-Type,Authorization`
- `POSTGRES_*` configures the service database for calendar connections and
  OAuth state rows.
- `JWT_ACCESS_SECRET` validates user JWTs on calendar OAuth endpoints.
- `INTERNAL_SERVICE_TOKEN` protects internal calendar event endpoints.
- `GOOGLE_CALENDAR_ENABLED` enables Calendar Sync v1.
- `CALENDAR_PROVIDER=mock|google`; `mock` is the local default, while `google`
  uses the real Google OAuth and Calendar APIs.
- `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, and
  `GOOGLE_OAUTH_REDIRECT_URL` are required for `CALENDAR_PROVIDER=google`.
- `GOOGLE_CALENDAR_SCOPES` defaults to
  `https://www.googleapis.com/auth/calendar.events`.
- `CALENDAR_TOKEN_ENCRYPTION_KEY` must be 16, 24, or 32 bytes and is used with
  AES-GCM. Do not use the dev value outside local development.
- `CALENDAR_OAUTH_STATE_TTL_SECONDS` defaults to `600`.
- `PUBLIC_WEB_BASE_URL` restricts OAuth callback redirects to the Web App.
- `DEFAULT_CALENDAR_TIMEZONE` defaults to `Europe/Bratislava`.

Documented for future providers, but unused in v1:

- `GOOGLE_PLACES_API_KEY`
- `MAPBOX_API_KEY`
- `OSRM_BASE_URL`
- `MAPBOX_ACCESS_TOKEN`
- `GOOGLE_MAPS_API_KEY`
- `OPEN_METEO_BASE_URL`
- `WEATHER_API_KEY`

Unsupported `PLACE_PROVIDER`, `ROUTE_PROVIDER`, `WEATHER_PROVIDER`,
`EXCHANGE_RATE_PROVIDER`, or `PRICE_PROVIDER` values fail startup with a clear
error. Providers are selected independently.

## Exchange Rates v1

The exchange-rate API is used by Trip Service budget summaries to convert item
and accommodation costs into the trip budget currency. The default `mock`
provider is deterministic and supports `EUR`, `USD`, `GBP`, `JPY`, `CZK`,
`PLN`, `HUF`, `CHF`, `CAD`, and `AUD`.

Examples:

```bash
curl "http://localhost:8084/exchange-rates/latest?base=EUR"
curl "http://localhost:8084/exchange-rates/convert?amount=2500&from=JPY&to=EUR"
```

Currency codes must be uppercase ISO-like 3-letter codes and amounts must be
non-negative. Identity conversion (`from == to`) returns provider `identity`,
rate `1`, and does not call a provider.

When a real provider adapter fails and fallback is enabled, the service logs a
safe warning and returns a mock result with `fallbackUsed: true`. If fallback is
disabled, the endpoint returns `502` with
`{"error":"exchange_rate_provider_unavailable"}`. Unsupported currencies return
`400` with `{"error":"unsupported_currency"}`.

Successful latest-rate tables are cached in memory by provider and base
currency. Errors are not cached, and the cache is cleared on service restart.

Limitations: conversions are approximate, no historical rates are supported, no
crypto rates are supported, and the API must not be used for financial advice or
trading.

## Attraction / Ticket Price API v1

`POST /prices/estimate` is an internal endpoint used by Trip Service price
enrichment after itinerary generation. It requires `X-Internal-Service-Token`
and returns either a provider `estimatedCost` or a no-match result. The Web App
does not call this endpoint directly.

Request:

```bash
curl -X POST "http://localhost:8084/prices/estimate" \
  -H "Content-Type: application/json" \
  -H "X-Internal-Service-Token: dev-internal-service-token" \
  -d '{
    "destination": "Rome",
    "currency": "EUR",
    "date": "2026-08-10",
    "place": {
      "provider": "mock",
      "providerPlaceId": "mock-colosseum",
      "name": "Colosseum",
      "category": "landmark",
      "lat": 41.8902,
      "lng": 12.4922
    },
    "itemContext": {
      "name": "Colosseum visit",
      "type": "attraction"
    }
  }'
```

Matched response:

```json
{
  "estimatedCost": {
    "amount": 19,
    "currency": "EUR",
    "category": "ticket",
    "confidence": "high",
    "source": "provider",
    "note": "Estimated entry ticket"
  },
  "provider": "mock",
  "fallbackUsed": false,
  "priceType": "ticket",
  "matched": true,
  "matchConfidence": 0.82,
  "metadata": { "reason": "Known mock attraction category" }
}
```

The mock provider is deterministic and approximate. It supports `EUR`, `USD`,
`GBP`, `CZK`, and `JPY`, returns no-match for likely free/public/non-ticket
items, and classifies likely paid museums, galleries, landmarks, towers, tours,
theme parks, palaces, castles, aquariums, and zoos into `ticket` or `activity`
costs. Unsupported currencies return `400` with `{"error":"unsupported_currency"}`.

`PRICE_PROVIDER=api` is a placeholder for a future real adapter. With fallback
enabled, startup and runtime API-provider failures fall back to the deterministic
mock provider; with fallback disabled, the endpoint returns `502` with
`{"error":"price_provider_unavailable"}`.

## Google Calendar Sync v1

User-facing OAuth endpoints require a valid Auth Service JWT, except
`GET /calendar/google/callback` because Google redirects the browser there with
`code` and `state`. The callback validates a single-use, expiring state row,
exchanges the authorization code server-side, fetches account email when
available, encrypts tokens at rest, and redirects only to URLs under
`PUBLIC_WEB_BASE_URL`.

Trip Service calls internal endpoints with `X-Internal-Service-Token`:

- `POST /internal/calendar/google/events/sync` creates or updates events in the
  user's primary Google calendar.
- `POST /internal/calendar/google/events/delete` removes events previously
  created by this app.

The real provider requests the least-privilege
`https://www.googleapis.com/auth/calendar.events` scope. Tokens, OAuth codes,
client secrets, and encryption keys must not be logged. The mock provider keeps
the same `/calendar/google/*` API but simulates OAuth and event operations for
local smoke tests.

Limitations: Google only, one connected account per user, primary calendar only,
one-way app-to-calendar sync, no watch/webhook subscriptions, no recurring
events, no Apple/Outlook integration, and no reminder customization.

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

The canonical place shape includes provider metadata, address, optional
coordinates, rating, category, website, a map URL, and optional
`openingHours`. Place metadata is optional on itinerary items and is persisted
by Trip Service as part of itinerary JSONB.

`openingHours` is an optional array of local-time intervals:

```json
{
  "openingHours": [
    { "dayOfWeek": 1, "open": "08:30", "close": "19:15" }
  ]
}
```

`dayOfWeek` uses `1 = Monday` through `7 = Sunday`. `open` and `close` use
24-hour `HH:mm` local time. Missing or empty `openingHours` means unknown;
closed all day is represented by no interval for that day. Multiple intervals
for the same day are allowed.

The mock provider includes deterministic places and simple opening hours for
Rome, Paris, Vienna, and Bratislava. Search is case-insensitive across place
name, category, and address. When a destination is provided, results are
filtered to that city. Unknown city-specific queries return a small fallback set
for that city.

Trip Service AI Place Enrichment v1 also calls `GET /places/search` after
itinerary generation. The endpoint returns normalized, map-ready place objects
with optional `category`, `rating`, `ratingCount`, `mapUrl`, `website`,
coordinates, and `openingHours` so Trip Service can attach high-confidence
matches without exposing provider credentials to the Web App.

## Real Place Provider v1

`PLACE_PROVIDER` options:

- `mock`: deterministic local data, no API key required. This is the default.
- `foursquare`: calls the Foursquare Places API and normalizes responses to the
  same `Place` JSON shape.

Foursquare configuration:

```bash
PLACE_PROVIDER=foursquare
FOURSQUARE_API_KEY=...
FOURSQUARE_BASE_URL=https://api.foursquare.com/v3
FOURSQUARE_TIMEOUT_SECONDS=8
PLACE_PROVIDER_FALLBACK_TO_MOCK=true
```

Search uses:

```text
GET /places/search?query={query}&near={destination}&limit=10
Authorization: <FOURSQUARE_API_KEY>
Accept: application/json
```

Details uses:

```text
GET /places/{fsq_id}
Authorization: <FOURSQUARE_API_KEY>
Accept: application/json
```

Foursquare fields are normalized as follows:

- `provider`: `foursquare`
- `providerPlaceId`: raw `fsq_id`
- `name`: Foursquare `name`
- `address`: `location.formatted_address` or a joined address fallback
- `latitude`/`longitude`: `geocodes.main`
- `rating`: Foursquare 0-10 rating converted to a 0-5 value
- `ratingCount`: `stats.total_ratings`
- `category`: first category name
- `website`: Foursquare `website`, when present
- `mapUrl`: provider URL when available, otherwise a Google Maps coordinate
  search URL when coordinates are available
- `openingHours`: empty in v1 for real provider results

Missing real-provider fields are valid. The Web App hides missing ratings,
ignores places without coordinates in map/distance features, and treats empty
`openingHours` as unknown.

When `PLACE_PROVIDER_FALLBACK_TO_MOCK=true`, startup can fall back to mock in
development if Foursquare configuration is missing, and runtime provider
request/response failures fall back to mock. Fallbacks are logged with
`fallbackUsed=true`, `provider=foursquare`, and `fallbackProvider=mock`. Disable
fallback with `PLACE_PROVIDER_FALLBACK_TO_MOCK=false` when provider failures
should surface as API errors.

Example:

```bash
curl "http://localhost:8084/places/search?query=Colosseum&destination=Rome"
```

## Routing API v1

`POST /routes/estimate` returns an approximate travel-time/distance estimate for
an ordered list of stops. The endpoint is provider-agnostic: a `mock` provider
(default and fallback) and a real `ors` ([OpenRouteService](https://openrouteservice.org))
provider sit behind a `RouteProvider` port, so the request/response contract is
identical regardless of which provider answers. The configured provider is never
exposed to the Web App beyond the `provider`/`fallbackUsed` fields in the
response, and API keys never leave the service.

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

How the **mock** provider estimates each consecutive stop pair:

- **Provider:** `mock`.
- **Distance:** Haversine straight-line distance × a `1.25` route factor, rounded
  to 2 decimals.
- **Speed:** flat per-mode pace — walking `5 km/h`, cycling `15 km/h`, driving
  `40 km/h`; `durationMinutes = round(distanceKm / speed * 60)`.
- **Totals:** `distanceKm` and `durationMinutes` are the sum of the segment
  values, so the total always equals the sum of the segments the caller sees.

The **ors** provider calls the OpenRouteService Directions v2 API:

- **Modes/profiles:** `walking → foot-walking`, `driving → driving-car`,
  `cycling → cycling-regular` (profiles are configurable).
- Coordinates are sent as `[longitude, latitude]` pairs (the order ORS requires);
  the API key is sent in the `Authorization` header and is never logged.
- Distance/duration come from the per-leg ORS `segments`; `routeGeometry` carries
  the encoded ORS polyline when available.
- HTTP failures are classified (401/403 → auth, 429 → rate limit, 5xx →
  unavailable, malformed body → bad response) and never surfaced verbatim.

**Provider selection** (`ROUTE_PROVIDER`):

- `mock` (default) → mock provider.
- `ors` → OpenRouteService, with mock as the fail-open fallback when
  `ROUTE_PROVIDER_FALLBACK_TO_MOCK=true`.
- Any other value fails fast at startup.
- If `ROUTE_PROVIDER=ors` but `ORS_API_KEY` is missing: fall back to mock when
  fallback is enabled, otherwise fail startup with a clear config error.

**Fallback:** when the real provider fails and fallback is enabled, the mock
provider answers and the response reports `"provider": "mock"` with
`"fallbackUsed": true`. When fallback is disabled, the endpoint returns
`502 {"error": "route_provider_unavailable"}`.

**Accepted modes** depend on the active provider: `mock` accepts `walking` only;
`ors` accepts `walking`, `driving`, and `cycling`.

Validation (returns `400` with `{"error": "..."}`):

- `mode` is required and must be supported by the active provider.
- `stops` is required, with a minimum of 2 and a maximum of 25 stops.
- each stop requires a `name` (≤ 200 chars) and valid coordinates
  (`latitude` ∈ [-90, 90], `longitude` ∈ [-180, 180]).

Limitations:

- The mock provider is not real routing — straight-line distance scaled by a
  constant factor, no traffic or elevation.
- ORS estimates depend on the configured profile; no Google Maps provider yet.
- The Web App uses these estimates read-only and falls back to its own Haversine
  straight-line estimate when the service is unavailable.

## Weather API v1

`GET /weather/forecast` returns daily forecasts for a destination and date range.
It is unauthenticated in v1. A `mock` provider (default and fallback) and a real
`openweathermap` ([OpenWeatherMap](https://openweathermap.org)) provider sit
behind a `WeatherProvider` port; the response contract is identical regardless of
which provider answers, and API keys never leave the service.

Request:

```bash
curl "http://localhost:8084/weather/forecast?destination=Rome&startDate=2026-08-10&days=3"
```

Response:

```json
{
  "destination": "Rome",
  "provider": "mock",
  "days": [
    {
      "date": "2026-08-10",
      "condition": "hot",
      "temperatureMinC": 24,
      "temperatureMaxC": 35,
      "precipitationChance": 5,
      "windSpeedKph": 10,
      "summary": "Hot and sunny",
      "warnings": [
        "High heat: avoid long outdoor walks at midday"
      ]
    }
  ]
}
```

Validation returns `400` with `{"error":"..."}`:

- `destination` is required and must be at most 200 characters.
- `startDate` is required in `YYYY-MM-DD` format.
- `days` is required and must be between 1 and 30.

Mock behavior:

- Deterministic: identical destination/start date/days returns identical data.
- Supports destination-aware patterns for Rome, Paris, Vienna, and Bratislava.
- Unknown destinations return reasonable generic seasonal forecasts.
- Rome in summer trends hotter and sunnier; Paris is milder with more rain;
  Vienna and Bratislava use moderate seasonal patterns.

The **openweathermap** provider:

- Geocodes the destination via the Geocoding API, then fetches the 5 day / 3 hour
  forecast and groups the 3-hour entries by **local** date (using the city's UTC
  offset).
- Per day: `temperatureMinC`/`temperatureMaxC` are the min/max across the day,
  `precipitationChance` is the max probability (`pop`), `windSpeedKph` is the max
  wind, and `condition`/`summary` come from the dominant condition. Temperatures
  and wind are normalized to Celsius / km/h regardless of `OPENWEATHER_UNITS`.
- **Coverage is all-or-nothing:** if any requested date is outside the provider's
  window (e.g. a trip more than ~5 days out), the provider reports an error and
  the mock fallback fills the whole forecast, so the response is always exactly
  `days` long with a single, honest `provider` label.
- HTTP failures are classified (401/403, 429, 5xx, malformed body, unknown
  destination) and never surfaced verbatim. The API key is sent as the `appid`
  query parameter and is never logged.

**Provider selection** (`WEATHER_PROVIDER`): `mock` (default) or `openweathermap`,
with the same opt-in / missing-key / fail-fast semantics as routing. When the
real provider fails and fallback is enabled, the response reports
`"provider": "mock"` with `"fallbackUsed": true`; when fallback is disabled, the
endpoint returns `502 {"error": "weather_provider_unavailable"}`.

Warnings:

- `temperatureMaxC >= 32`: `High heat: avoid long outdoor walks at midday`
- `precipitationChance >= 60`: `Rain likely: consider indoor alternatives`
- `windSpeedKph >= 35`: `Windy: viewpoints and exposed areas may be uncomfortable`
- `temperatureMaxC <= 5`: `Cold day: plan warm indoor breaks`

Limitations:

- In-memory cache only (no Redis); cleared on restart.
- OpenWeatherMap free tier covers ~5 days; out-of-range dates use the mock
  fallback.
- Destination geocoding may be approximate.
- Provider rate limits apply.

## Provider Caching

Route, weather, exchange-rate, and price responses are cached in a small,
process-local TTL cache to cut repeated upstream calls:

- **In-memory only** — concurrency-safe, no Redis, no database; cleared on restart.
- **Route cache key:** `route:<provider>:<mode>:<lat,lng|...>` with coordinates
  rounded to 5 decimals (`ROUTE_CACHE_TTL_SECONDS`, default 6h).
- **Weather cache key:** `weather:<provider>:<destination>:<startDate>:<days>:<units>`
  (`WEATHER_CACHE_TTL_SECONDS`, default 1h). A weather cache hit skips both the
  geocoding and forecast upstream calls.
- **Exchange-rate cache key:** `exchange-rate:<provider>:<base>`
  (`EXCHANGE_RATE_CACHE_TTL_SECONDS`, default 6h).
- **Price cache key:** `price:<provider>:<destination>:<currency>:<date>:<place>`
  with provider/id/name/category context (`PRICE_CACHE_TTL_SECONDS`, default
  24h). No-match results are cached too because they represent a successful
  provider answer.
- Only successful, non-fallback results are cached; provider errors and fallback
  results are never cached, so a transient outage is retried rather than pinned.
- Each lookup logs `cacheHit`, `provider`, and the endpoint type.

## Limitations

- In-memory cache only (no Redis); provider rate limits still apply.
- ORS route estimates depend on the configured profile; no Google Maps yet.
- OpenWeatherMap free tier covers ~5 days; out-of-range trip dates use the mock
  fallback, and destination geocoding may be approximate.
- Price estimates are approximate ticket/activity hints only; no booking,
  checkout, inventory, or guaranteed real-time attraction pricing is supported.
- The mock providers remain the default and the fail-open fallback.
