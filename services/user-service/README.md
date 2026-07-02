# User Service

User/Profile Service v1 owns authenticated users' travel profiles and travel
preferences. Auth Service owns identity and token issuance; User Service only
validates Auth Service JWT access tokens locally with the shared
`JWT_ACCESS_SECRET`.

## Endpoints

Public:

- `GET /health`
- `GET /ready`

Protected:

- `GET /users/me/profile`
- `PUT /users/me/profile`
- `GET /users/me/preferences`
- `PATCH /users/me/preferences`

Protected endpoints require `Authorization: Bearer <accessToken>`. The service
uses the JWT `sub` claim as the owner user ID; clients must not send `userId`.
During itinerary generation, Trip Service calls these same protected endpoints
with the authenticated user's JWT to load trusted profile/preferences for AI
Planning Service personalization. User Service should continue to avoid logging
access tokens or sensitive preference payloads.

## Environment

- `APP_ENV=development`
- `HTTP_ADDRESS=:8083`
- `POSTGRES_DB=user_service`
- `POSTGRES_USER=postgres`
- `POSTGRES_PASSWORD=postgres`
- `POSTGRES_HOST=localhost`
- `POSTGRES_PORT=5432`
- `POSTGRES_MIG_PATH=./migrations`
- `JWT_ACCESS_SECRET=change-me-in-development`
- `CORS_ALLOWED_ORIGINS=http://localhost:3000`

In production, `JWT_ACCESS_SECRET` must not be the development default and must
be at least 32 characters.

## Local Commands

```bash
cp .env.example .env
set -a; source .env; set +a
make run
```

The service applies golang-migrate migrations on startup. To run them manually:

```bash
make migrate-up
```

This service follows the auth/trip service repository style and uses squirrel
query builders rather than sqlc.

## Curl Examples

```bash
TOKEN="<access token>"

curl -H "Authorization: Bearer ${TOKEN}" \
  http://localhost:8083/users/me/profile

curl -X PUT http://localhost:8083/users/me/profile \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "Dmytro",
    "homeCity": "Bratislava",
    "homeCountry": "Slovakia",
    "preferredCurrency": "EUR",
    "preferredLanguage": "en"
  }'

curl -H "Authorization: Bearer ${TOKEN}" \
  http://localhost:8083/users/me/preferences

curl -X PATCH http://localhost:8083/users/me/preferences \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "travelStyles": ["budget", "food", "hidden_gems"],
    "pace": "balanced",
    "maxWalkingKmPerDay": 8,
    "foodPreferences": ["local", "cheap"],
    "avoid": ["nightclubs"],
    "preferredTransport": ["walking", "public_transport"],
    "accommodationStyle": ["budget_hotel"]
  }'
```

## Observability

- `GET /metrics` exposes Prometheus metrics.
- HTTP middleware records `http_requests_total`,
  `http_request_duration_seconds`, and `http_requests_in_flight`.
- User counters include `user_profile_requests_total` and
  `user_preferences_requests_total`, labeled by bounded operation/result values.
- The service reads or generates `X-Request-ID` and `X-Correlation-ID`, echoes
  them on responses, and includes them in request logs.
- Do not log Authorization headers, full travel preferences, or private profile
  payloads.
