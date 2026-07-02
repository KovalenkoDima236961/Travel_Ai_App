# Auth Service

Auth Service v1 is a Go microservice for first-party email/password
authentication in the Travel AI App. It supports registration, login, refresh
token rotation, logout, and current-user lookup with JWT access tokens.

## Endpoints

| Method | Path             | Description |
| ------ | ---------------- | ----------- |
| GET    | `/health`        | Liveness check. |
| GET    | `/ready`         | PostgreSQL readiness check. |
| POST   | `/auth/register` | Create a user and issue tokens. |
| POST   | `/auth/login`    | Verify credentials and issue tokens. |
| POST   | `/auth/refresh`  | Rotate refresh token and issue a new access token. |
| POST   | `/auth/logout`   | Revoke a refresh token if present. |
| GET    | `/auth/me`       | Return the user from a Bearer access token. |

## Environment

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `APP_ENV` | `development` | `development` or `production`. |
| `HTTP_ADDRESS` | `:8082` | HTTP listen address. |
| `POSTGRES_DB` | — | PostgreSQL database name. |
| `POSTGRES_USER` | — | PostgreSQL user. |
| `POSTGRES_PASSWORD` | — | PostgreSQL password. |
| `POSTGRES_HOST` | — | PostgreSQL host. |
| `POSTGRES_PORT` | — | PostgreSQL port. |
| `POSTGRES_MIN_CONNS` | — | Pool minimum connections. |
| `POSTGRES_MAX_CONNS` | — | Pool maximum connections. |
| `POSTGRES_MIG_PATH` | `./migrations` | Path to golang-migrate files. |
| `JWT_ACCESS_SECRET` | `change-me-in-development` | HS256 signing secret. Required; production rejects the default and secrets shorter than 32 characters. |
| `ACCESS_TOKEN_TTL_MINUTES` | `15` | Access token lifetime. |
| `REFRESH_TOKEN_TTL_DAYS` | `30` | Refresh token lifetime. |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated browser origins. |
| `CORS_ALLOWED_METHODS` | `GET,POST,OPTIONS` | CORS preflight methods. |
| `CORS_ALLOWED_HEADERS` | `Content-Type,Authorization` | CORS preflight headers. |

## Run Locally

From `services/auth-service`:

```bash
export APP_ENV=development \
  HTTP_ADDRESS=:8082 \
  POSTGRES_DB=auth_service \
  POSTGRES_USER=postgres \
  POSTGRES_PASSWORD=postgres \
  POSTGRES_HOST=localhost \
  POSTGRES_PORT=5432 \
  POSTGRES_MIN_CONNS=2 \
  POSTGRES_MAX_CONNS=10 \
  POSTGRES_MIG_PATH=./migrations \
  JWT_ACCESS_SECRET=change-me-in-development

go run ./cmd/server
```

Or run with a YAML config file, matching the Trip Service bootstrap style:

```bash
cp configs/config.example.yaml configs/config.yaml
go run ./cmd/server -config ./configs/config.yaml
```

Migrations run automatically on startup. To run them manually:

```bash
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/auth_service?sslmode=disable" up
```

PostgreSQL queries are built in `internal/infrastructure/repository/postgres`
using squirrel, matching the trip-service repository style.

## Example Curl

Register:

```bash
curl -sS -X POST http://localhost:8082/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"StrongPassword123!"}'
```

Login:

```bash
curl -sS -X POST http://localhost:8082/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"StrongPassword123!"}'
```

Me:

```bash
curl -sS http://localhost:8082/auth/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Refresh:

```bash
curl -sS -X POST http://localhost:8082/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}"
```

Logout:

```bash
curl -sS -X POST http://localhost:8082/auth/logout \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}"
```

## Observability

- `GET /metrics` exposes Prometheus metrics.
- HTTP middleware records `http_requests_total`,
  `http_request_duration_seconds`, and `http_requests_in_flight`.
- Auth counters include `auth_register_total`, `auth_login_total`,
  `auth_refresh_total`, and `auth_logout_total`, each labeled only by bounded
  result values.
- The service reads or generates `X-Request-ID` and `X-Correlation-ID`, echoes
  them on responses, and includes them in request logs.
- Do not log Authorization headers, refresh tokens, password values, cookies, or
  full request bodies.
