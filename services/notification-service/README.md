# Notification Service

In-app notification service for the Travel AI Planner. It stores and serves
private, per-user notifications ("a collaborator commented on your trip", "you
were invited to collaborate", …) and is written/read over plain HTTP.

It is intentionally small and replaceable: Trip Service calls it **synchronously**
after a successful action, and the web app **polls** it for the unread count.
There is no message broker, no WebSocket, no email/push — see [Limitations](#limitations).

## Architecture

Same stack and layout as the other Go services in this repo (Auth/Trip):

- `net/http` + `go-chi/chi/v5`, hand-wired composition root in `internal/app`
  (no DI framework).
- `squirrel` query building over a `pgx` pool (`pkg/storage/postgres`); migrations
  applied automatically on startup via `golang-migrate`.
- Config via `cleanenv` (YAML file or env), validated with `go-playground/validator`.
- Structured logging with `zap`; graceful shutdown via the LIFO `pkg/closer`.
- Layered: `domain` (entities/errors) → `notifications` (use cases) →
  `infrastructure/repository/postgres` (adapter) → `http-server` (transport).

## Endpoints

### Health
| Method | Path      | Auth | Description |
|--------|-----------|------|-------------|
| GET    | `/health` | none | Liveness. Always 200 when the process is up. |
| GET    | `/ready`  | none | Readiness. 200 only when Postgres is reachable. |

### User-facing (require a valid Auth Service access token)
All routes derive `user_id` from the token `sub` claim, so a user can only ever
see their own notifications.

| Method | Path                          | Description |
|--------|-------------------------------|-------------|
| GET    | `/notifications?limit=&cursor=` | Current user's notifications, newest first, cursor-paginated. |
| GET    | `/notifications/unread-count`   | `{ "count": N }` of unread notifications. |
| PATCH  | `/notifications/{id}/read`      | Mark one notification read (idempotent). |
| PATCH  | `/notifications/read-all`       | Mark all of the user's unread notifications read. |

`limit` defaults to 30, max 100. `cursor` is the opaque `nextCursor` from a prior
list response.

### Internal (require `X-Internal-Service-Token`, no user JWT)
For the private service network only — never exposed to browsers.

| Method | Path                            | Description |
|--------|---------------------------------|-------------|
| POST   | `/internal/notifications/batch` | Create up to 100 notifications. Skips any where `userId == actorUserId`. Returns `{ "created": N }`. |

Example batch request:

```json
{
  "notifications": [
    {
      "userId": "recipient-user-id",
      "tripId": "trip-id",
      "actorUserId": "actor-user-id",
      "type": "comment_created",
      "title": "New comment",
      "message": "A collaborator commented on Day 2 · Louvre Museum.",
      "entityType": "comment",
      "entityId": "comment-id",
      "metadata": { "dayNumber": 2, "itemIndex": 3, "itemName": "Louvre Museum" }
    }
  ]
}
```

## Database

One table, `notifications`, in its own logical database (`notification_service`).
Migrations live in `migrations/` and run on startup.

| Column          | Type        | Notes |
|-----------------|-------------|-------|
| `id`            | UUID PK     | `gen_random_uuid()` |
| `user_id`       | UUID NOT NULL | recipient |
| `trip_id`       | UUID NULL   | related trip, when applicable |
| `actor_user_id` | UUID NULL   | who triggered it |
| `type`          | TEXT NOT NULL | one of the types below |
| `title`         | TEXT NOT NULL | ≤ 200 chars (CHECK) |
| `message`       | TEXT NOT NULL | ≤ 1000 chars (CHECK) |
| `entity_type`   | TEXT NULL   | `trip`/`comment`/`collaborator`/`itinerary`/… |
| `entity_id`     | UUID NULL   | target entity id |
| `metadata`      | JSONB NOT NULL DEFAULT `'{}'` | rendering hints; **never secrets** |
| `read_at`       | TIMESTAMP NULL | null = unread |
| `created_at`    | TIMESTAMP NOT NULL DEFAULT NOW() | |

Indexed on `user_id`, `(user_id, read_at)`, `(user_id, created_at DESC)`,
`trip_id`, `type`, and `(entity_type, entity_id)`.

## Notification types

`collaboration_invited`, `collaboration_accepted`, `collaborator_role_changed`,
`collaborator_removed`, `comment_created`, `itinerary_updated`,
`itinerary_generated`, `day_regenerated`, `item_regenerated`, `version_restored`.

The internal endpoint **rejects unknown types** so a caller typo never lands an
un-renderable notification in someone's inbox.

## Authentication

- **User endpoints** validate the same HS256 access token issued by Auth Service
  (`JWT_ACCESS_SECRET`), checking signature, expiry, and `sub`. Unlike Trip
  Service there is **no development fallback identity** — a missing/invalid token
  is always 401, because notifications are private user data.
- **Internal endpoint** requires `X-Internal-Service-Token` to equal
  `INTERNAL_SERVICE_TOKEN` (constant-time compare). It trusts the caller to
  supply recipient ids and does not require a user JWT. This is a deliberately
  simple v1 scheme, replaceable later by mTLS / signed service tokens / an event
  bus without changing callers.

## Configuration

Loaded from a YAML file (`-config ./configs/config.yaml`) or environment.

| Env | Default | Description |
|-----|---------|-------------|
| `APP_ENV` | `development` | `development` or `production` |
| `HTTP_ADDRESS` | `:8086` | Listen address |
| `JWT_ACCESS_SECRET` | `change-me-in-development` | Must match Auth Service |
| `JWT_ISSUER` / `JWT_AUDIENCE` | empty | Optional; reserved for stricter validation |
| `AUTH_HEADER_NAME` | `Authorization` | Bearer token header |
| `INTERNAL_SERVICE_TOKEN` | `dev-internal-service-token` | Shared service token |
| `POSTGRES_DB` | `notification_service` | Database name |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | `postgres` | Credentials |
| `POSTGRES_HOST` / `POSTGRES_PORT` | `localhost` / `5432` | Host/port |
| `POSTGRES_MIN_CONNS` / `POSTGRES_MAX_CONNS` | `2` / `10` | Pool sizing |
| `POSTGRES_MIG_PATH` | `./migrations` | Migrations directory |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Browser origins |

In production the service refuses to start with the default JWT secret or the
default internal token.

## Development

```bash
make run           # run with env config
make config-run    # run with ./configs/config.yaml
make test          # go test ./... -race
make lint          # golangci-lint
```

## Limitations (v1)

- No email or push notifications.
- No WebSockets / Server-Sent Events — the web app polls the unread count.
- No RabbitMQ / Kafka / event bus — Trip Service calls this service
  **synchronously over HTTP** after a successful action, fail-open (a
  notification failure never breaks the originating Trip Service action).
- No background workers.

These are intentional and documented so the synchronous HTTP design can be
swapped for an event bus later without changing the public API.
