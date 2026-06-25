# Notification Service

In-app notification service for the Travel AI Planner. It stores and serves
private, per-user notifications ("a collaborator commented on your trip", "you
were invited to collaborate", …) and is written/read over plain HTTP.

It is intentionally small and replaceable: Trip Service calls it **synchronously**
after a successful action, and the web app **polls** it for the unread count.
There is no message broker and no WebSocket. It can **optionally send email** for
selected notification types after the in-app rows are created — see
[Email notifications (v1)](#email-notifications-v1) and [Limitations](#limitations).

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
| POST   | `/internal/notifications/batch` | Create up to 100 notifications. Skips any where `userId == actorUserId`. Then fans out email for allowlisted types. Returns `{ "created": N, "email": { "attempted", "sent", "skipped", "failed" } }`. |

Example response:

```json
{
  "created": 3,
  "email": { "attempted": 2, "sent": 2, "skipped": 1, "failed": 0 }
}
```

In-app rows are created **first** and are never rolled back because of an email
failure. When email is fail-open (or disabled) a send failure is reported only in
`email.failed` with HTTP 201; when email is fail-closed and a send fails the rows
still exist but the endpoint returns **502** so the caller can observe the
degraded delivery.

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

## Email notifications (v1)

After in-app rows are created, the internal batch handler hands the created
notifications to the email orchestration (`internal/emailnotifications`), which:

1. **Filters** by policy — email must be enabled, the type must be in the
   allowlist, and the recipient must not be the actor.
2. **Resolves** recipient emails in one batch call to Auth Service
   (`POST /internal/users/batch`), authenticated with `INTERNAL_SERVICE_TOKEN`.
   Auth Service owns email in v1; `displayName` is currently empty and templates
   fall back to a neutral greeting.
3. **Builds** a short email (`internal/emailnotifications/templates.go`) and
   **sends** it via the configured provider (`internal/email`).

Sending is **synchronous** inside the request — there is no queue or worker yet.

### Providers

- `EMAIL_PROVIDER=mock` (default) — sends no external mail; logs masked
  recipient + subject only. Safe for local dev with email enabled.
- `EMAIL_PROVIDER=smtp` — sends via `net/smtp`. Requires `SMTP_HOST` and
  `SMTP_FROM_EMAIL`; uses auth when `SMTP_USERNAME` is set; negotiates STARTTLS
  when the server advertises it (use a STARTTLS port such as 587 — implicit TLS
  on 465 is not supported in v1). An unsupported provider is a **startup error**.

### Allowlisted types

`EMAIL_NOTIFICATION_TYPES` (comma-separated). Default:
`collaboration_invited`, `comment_created`, `collaborator_role_changed`,
`collaborator_removed`. Other types (`collaboration_accepted`, `day_regenerated`,
`item_regenerated`, `version_restored`, `itinerary_updated`,
`itinerary_generated`) have templates and can be enabled by adding them to the
allowlist.

### Fail-open behavior

`EMAIL_NOTIFICATIONS_FAIL_OPEN=true` (default): recipient-lookup or send failures
are logged and surfaced in the response stats, but the request still returns 201.
`=false`: a failure returns 502 **after** the in-app rows are committed (they are
never rolled back).

### Privacy

Emails never contain secrets, JWTs, share access tokens, share passwords, full
itinerary payloads, private preferences, or comment bodies (only day/item).
`SMTP_PASSWORD` is never logged and bodies are never logged at info level;
recipient addresses are masked in logs (`an***@example.com`).

## Configuration

Loaded from a YAML file (`-config ./configs/config.yaml`) or environment.

| Env | Default | Description |
|-----|---------|-------------|
| `APP_ENV` | `development` | `development` or `production` |
| `HTTP_ADDRESS` | `:8086` | Listen address |
| `JWT_ACCESS_SECRET` | `change-me-in-development` | Must match Auth Service |
| `JWT_ISSUER` / `JWT_AUDIENCE` | empty | Optional; reserved for stricter validation |
| `AUTH_HEADER_NAME` | `Authorization` | Bearer token header |
| `INTERNAL_SERVICE_TOKEN` | `dev-internal-service-token` | Shared service token (incoming batch + outgoing user lookup) |
| `EMAIL_NOTIFICATIONS_ENABLED` | `true` | Enable/disable email globally |
| `EMAIL_NOTIFICATIONS_FAIL_OPEN` | `true` | Log email errors instead of failing the batch |
| `EMAIL_PROVIDER` | `mock` | `mock` or `smtp` |
| `EMAIL_NOTIFICATION_TYPES` | `collaboration_invited,comment_created,collaborator_role_changed,collaborator_removed` | Allowlist of types that trigger email |
| `PUBLIC_WEB_BASE_URL` | `http://localhost:3000` | Used to build safe links back to the Web App |
| `AUTH_SERVICE_URL` | `http://auth-service:8082` | Service that owns recipient email (v1) |
| `USER_SERVICE_URL` | `http://user-service:8083` | Reserved for future display-name enrichment |
| `USER_LOOKUP_TIMEOUT_SECONDS` | `5` | Recipient lookup timeout |
| `SMTP_HOST` | empty | SMTP host (required when provider=smtp) |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | empty | SMTP auth (used when username set); password is never logged |
| `SMTP_FROM_EMAIL` | `no-reply@localhost` | From address (required when provider=smtp) |
| `SMTP_FROM_NAME` | `AI Travel Planner` | From display name |
| `SMTP_USE_TLS` | `true` | Reserved/STARTTLS hint |
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

- **Email is synchronous** inside the batch request — no retry queue and no
  background worker. A slow SMTP server slows the request.
- No push notifications.
- No per-user notification/email preferences and no unsubscribe/preferences page.
- No email digests.
- `mock` provider is the local-dev default and sends no external mail.
- No WebSockets / Server-Sent Events — the web app polls the unread count.
- No RabbitMQ / Kafka / event bus — Trip Service calls this service
  **synchronously over HTTP** after a successful action, fail-open (a
  notification failure never breaks the originating Trip Service action).
- No background workers.

These are intentional and documented so the synchronous HTTP design can be
swapped for an event bus / async worker later without changing the public API.
