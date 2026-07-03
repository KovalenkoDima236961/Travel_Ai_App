# Notification Service

In-app notification service for the Travel AI Planner. It stores and serves
private, per-user notifications ("a collaborator commented on your trip", "you
were invited to collaborate", …) and is written/read over plain HTTP.

It is intentionally small and replaceable: Trip Service calls it **synchronously**
after a successful action, and the web app opens an authenticated
Server-Sent Events stream for new in-app notifications while keeping polling as
a fallback. There is no message broker and no WebSocket. It can **optionally send
email** for selected notification types and browser Web Push notifications for
important events after the in-app rows are created — see
[Email notifications (v1)](#email-notifications-v1),
[Web Push notifications (v1)](#web-push-notifications-v1), and
[Limitations](#limitations).

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

### User-facing (require a valid Auth Service access token except public key)
Authenticated routes derive `user_id` from the token `sub` claim, so a user can
only ever see or change their own notifications and push subscriptions.

| Method | Path                          | Description |
|--------|-------------------------------|-------------|
| GET    | `/notifications?limit=&cursor=` | Current user's notifications, newest first, cursor-paginated. |
| GET    | `/notifications/unread-count`   | `{ "count": N }` of unread notifications. |
| GET    | `/notifications/stream`         | Authenticated SSE stream for real-time notification updates. |
| GET    | `/notifications/push/public-key` | VAPID public key and enabled state. No JWT required. |
| POST   | `/notifications/push/subscribe`  | Store or refresh the current browser's push subscription. |
| DELETE | `/notifications/push/unsubscribe` | Disable the current user's push subscription endpoint. |
| GET    | `/notifications/push/status`     | Push enabled state and active subscription count. |
| GET    | `/notifications/preferences`     | Full effective in-app/email/push preference matrix for the current user. |
| PUT    | `/notifications/preferences`     | Upsert the current user's notification preferences. |
| PATCH  | `/notifications/{id}/read`      | Mark one notification read (idempotent). |
| PATCH  | `/notifications/read-all`       | Mark all of the user's unread notifications read. |

`limit` defaults to 30, max 100. `cursor` is the opaque `nextCursor` from a prior
list response.

### Real-time notifications (v1)

`GET /notifications/stream` is an authenticated Server-Sent Events endpoint. The
client sends the same `Authorization: Bearer <accessToken>` header used for the
other user-facing notification routes.

The stream emits:

- `notification.created` after an in-app notification row has been created in
  the database and preference filtering has allowed it.
- `heartbeat` periodically, and once on connect with `{ "status": "connected" }`.

`notification.created` uses the same notification DTO shape as
`GET /notifications`:

```text
event: notification.created
data: {"notification":{"id":"...","userId":"...","tripId":"...","actorUserId":"...","type":"comment_created","title":"New comment","message":"...","entityType":"comment","entityId":"...","metadata":{},"readAt":null,"createdAt":"..."}}
```

The stream manager is in-memory and supports multiple active tabs/devices per
user, capped by `NOTIFICATION_SSE_MAX_CONNECTIONS_PER_USER`. If the cap is
exceeded, the new stream request returns `429`. If a connected client falls
behind and its event queue fills, the event is dropped for that client only; the
notification row remains recoverable through the list endpoint and polling.

If a reverse proxy is placed in front of this service, disable response buffering
for `/notifications/stream` and keep the connection alive. The service sets
`X-Accel-Buffering: no`, but proxy configuration may still be required.

### Internal (require `X-Internal-Service-Token`, no user JWT)
For the private service network only — never exposed to browsers.

| Method | Path                            | Description |
|--------|---------------------------------|-------------|
| POST   | `/internal/notifications/batch` | Create up to 100 notifications. Skips any where `userId == actorUserId`, applies in-app preferences, then fans out email and push independently for allowlisted/preference-enabled types. |

Example response:

```json
{
  "requested": 5,
  "created": 3,
  "skipped": 2,
  "skippedByPreference": 2,
  "email": {
    "attempted": 1,
    "sent": 1,
    "skipped": 4,
    "skippedByPreference": 3,
    "failed": 0
  },
  "push": {
    "attempted": 2,
    "sent": 2,
    "skipped": 3,
    "skippedByPreference": 1,
    "failed": 0,
    "subscriptionsDisabled": 0
  }
}
```

In-app rows are created **first** and are never rolled back because of an email
failure. Email is evaluated separately from in-app preferences: a user can
disable in-app comments while keeping comment emails enabled, or the reverse.
Push is also evaluated as an independent channel. Expired or invalid push
subscriptions are soft-disabled automatically.
When email is fail-open (or disabled) a send failure is reported only in
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

The service owns its own logical database (`notification_service`). Migrations
live in `migrations/` and run on startup.

### `notifications`

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

### `notification_preferences`

Stores sparse per-user overrides for future notifications. Missing rows mean
"use defaults"; existing notification rows are never modified.

| Column       | Type          | Notes |
|--------------|---------------|-------|
| `id`         | UUID PK       | `gen_random_uuid()` |
| `user_id`    | UUID NOT NULL | owner of the preference |
| `channel`    | TEXT NOT NULL | `in_app`, `email`, or `push` |
| `category`   | TEXT NOT NULL | `collaboration`, `comments`, `trip_updates`, or `role_changes` |
| `enabled`    | BOOLEAN NOT NULL | channel/category state |
| `created_at` | TIMESTAMP NOT NULL DEFAULT NOW() | |
| `updated_at` | TIMESTAMP NOT NULL DEFAULT NOW() | updated on upsert |

Constrained by `UNIQUE (user_id, channel, category)` and indexed on `user_id`
and `(user_id, channel)`.

### `push_subscriptions`

Stores one row per browser Push API subscription. Multiple active subscriptions
per user are supported.

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID PK | `gen_random_uuid()` |
| `user_id` | UUID NOT NULL | owner |
| `endpoint` | TEXT NOT NULL UNIQUE | push-service endpoint; do not log in full |
| `p256dh` / `auth` | TEXT NOT NULL | browser key material; do not log |
| `user_agent` / `browser` / `device_label` | TEXT NULL | device metadata |
| `status` | TEXT NOT NULL | `active` or `disabled` |
| `created_at` / `updated_at` | TIMESTAMP NOT NULL | |
| `last_used_at` | TIMESTAMP NULL | last successful send |
| `disabled_at` / `disable_reason` | TIMESTAMP/TEXT NULL | cleanup/audit state |

Indexed on `user_id`, `status`, and `(user_id, status)`.

## Notification types

`collaboration_invited`, `collaboration_accepted`, `collaborator_role_changed`,
`collaborator_removed`, `comment_created`, `itinerary_updated`,
`itinerary_generated`, `day_regenerated`, `item_regenerated`,
`version_restored`, `generation_job_failed`, `budget_optimization_ready`,
`budget_optimization_failed`.

Unknown types are accepted for forward compatibility and use the preference
fallbacks documented below: in-app allowed, email blocked.

## Notification preferences (v1)

Preferences are global per user, category-based, and apply only to future
notifications.

Channels:

- `in_app`
- `email`
- `push`

Categories and type mapping:

- `collaboration`: `collaboration_invited`, `collaboration_accepted`
- `comments`: `comment_created`
- `role_changes`: `collaborator_role_changed`, `collaborator_removed`
- `trip_updates`: `itinerary_updated`, `itinerary_generated`,
  `day_regenerated`, `item_regenerated`, `version_restored`,
  `generation_job_failed`, `budget_optimization_ready`,
  `budget_optimization_failed`

Defaults for a user with no stored rows:

| Category | In-app | Email | Push |
|----------|--------|-------|------|
| `collaboration` | enabled | enabled | enabled |
| `comments` | enabled | enabled | enabled |
| `role_changes` | enabled | enabled | enabled |
| `trip_updates` | enabled | disabled | enabled |

`GET /notifications/preferences` returns all 12 channel/category combinations
after merging stored rows over these defaults:

```json
{
  "items": [
    { "channel": "in_app", "category": "collaboration", "enabled": true },
    { "channel": "email", "category": "trip_updates", "enabled": false },
    { "channel": "push", "category": "trip_updates", "enabled": true }
  ]
}
```

`PUT /notifications/preferences` accepts up to 20 items and upserts each
channel/category pair for the authenticated user. `enabled` is required, unknown
channels/categories are rejected with 400, and duplicate pairs in one request are
rejected. The request never accepts a user id; the user comes from the JWT
subject.

Preferences do not affect core app data: collaboration invitation records,
comments, collaborator roles, activity feed rows, and existing notifications are
unchanged. If in-app collaboration invitations are disabled, the invitation still
exists in the Trips page invitation flow; only the notification row is skipped.
Unknown notification types are allowed in-app by default and are not emailed.
Unknown notification types are also not pushed until they have an explicit
lock-screen-safe payload policy.

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

After the internal batch is validated, the handler hands non-self email
candidates to the email orchestration (`internal/emailnotifications`), which:

1. **Filters** by policy — email must be enabled, the type must be in the
   allowlist, the type must map to a preference category, the recipient's email
   preference for that category must be enabled, and the recipient must not be
   the actor.
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

### Allowlisted types and preferences

`EMAIL_NOTIFICATION_TYPES` (comma-separated). Default:
`collaboration_invited`, `comment_created`, `collaborator_role_changed`,
`collaborator_removed`. Other types (`collaboration_accepted`, `day_regenerated`,
`item_regenerated`, `version_restored`, `itinerary_updated`,
`itinerary_generated`) have templates and can be enabled by adding them to the
allowlist.

The allowlist and user preferences are both required. If a type is allowlisted
but the recipient disabled email for its category, the email is skipped and
counted in `email.skippedByPreference`.

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

## Web Push notifications (v1)

Browser push uses VAPID and the standard Push API. There is no Firebase Cloud
Messaging, APNS, native mobile push, SMS, or paid push vendor.

User flow:

1. The web app reads `GET /notifications/push/public-key`.
2. After an explicit user click, the browser asks for notification permission,
   registers `/sw.js`, and subscribes `PushManager` with the VAPID public key.
3. `POST /notifications/push/subscribe` stores the endpoint/key material for the
   authenticated user. Re-subscribing the same endpoint refreshes the keys,
   metadata, and active status.
4. Internal notification batches evaluate push preferences independently from
   in-app and email preferences, then send to all active subscriptions.
5. `404`/`410` and invalid-subscription responses disable the subscription so it
   is not retried forever.
6. `DELETE /notifications/push/unsubscribe` soft-disables the current user's
   endpoint and succeeds even when the row is already absent.

Only selected lock-screen-safe types are pushed by default:
`collaboration_invited`, `collaboration_accepted`,
`collaborator_role_changed`, `collaborator_removed`, `comment_created`,
`itinerary_generated`, `generation_job_failed`, `budget_optimization_ready`, and
`budget_optimization_failed`. Payloads are short and contain no full comments,
full itineraries, tokens, private notes, OAuth data, or API keys. Click URLs are
relative app URLs such as `/trips/{tripId}` or `/notifications`.

Generate local VAPID keys with:

```bash
npx web-push generate-vapid-keys
```

Then set:

```bash
WEB_PUSH_ENABLED=true
WEB_PUSH_VAPID_PUBLIC_KEY=...
WEB_PUSH_VAPID_PRIVATE_KEY=...
WEB_PUSH_SUBJECT=mailto:dev@example.com
```

The public key is safe to expose to browsers. The private key is a secret and
must not be committed or logged. Online VAPID key generators are not recommended
for real environments.

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
| `NOTIFICATION_SSE_ENABLED` | `true` | Enable authenticated SSE stream |
| `NOTIFICATION_SSE_HEARTBEAT_SECONDS` | `25` | Heartbeat interval |
| `NOTIFICATION_SSE_WRITE_TIMEOUT_SECONDS` | `10` | Per-event stream write timeout |
| `NOTIFICATION_SSE_MAX_CONNECTIONS_PER_USER` | `5` | Active SSE connections allowed per user on this instance |
| `SMTP_HOST` | empty | SMTP host (required when provider=smtp) |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | empty | SMTP auth (used when username set); password is never logged |
| `SMTP_FROM_EMAIL` | `no-reply@localhost` | From address (required when provider=smtp) |
| `SMTP_FROM_NAME` | `AI Travel Planner` | From display name |
| `SMTP_USE_TLS` | `true` | Reserved/STARTTLS hint |
| `WEB_PUSH_ENABLED` | `false` | Enable browser Web Push when VAPID keys are configured |
| `WEB_PUSH_VAPID_PUBLIC_KEY` | empty | Browser-visible VAPID public key |
| `WEB_PUSH_VAPID_PRIVATE_KEY` | empty | Secret VAPID private key |
| `WEB_PUSH_SUBJECT` | `mailto:support@example.com` | VAPID subject |
| `WEB_PUSH_TIMEOUT_SECONDS` | `8` | Per-subscription send timeout |
| `WEB_PUSH_TTL_SECONDS` | `3600` | Push message TTL |
| `WEB_PUSH_URGENCY` | `normal` | Web Push urgency (`very-low`, `low`, `normal`, `high`) |
| `WEB_PUSH_FAIL_OPEN` | `true` | Log push errors instead of failing the batch |
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

## Observability

- `GET /metrics` exposes Prometheus metrics.
- HTTP middleware records `http_requests_total`,
  `http_request_duration_seconds`, and `http_requests_in_flight`.
- Notification metrics include `notifications_created_total`,
  `notifications_failed_total`, `notifications_email_sent_total`,
  `push_notifications_sent_total`, `push_notifications_failed_total`,
  `push_subscriptions_disabled_total`,
  `notifications_sse_connections`, `notifications_sse_events_sent_total`, and
  `notifications_sse_events_dropped_total`.
- The service reads or generates `X-Request-ID` and `X-Correlation-ID`, echoes
  them on responses, includes them in logs, and propagates them on internal
  Auth/User Service calls.
- Do not log access tokens, internal service tokens, SMTP credentials, recipient
  payloads beyond safe IDs/counts, or full notification metadata.

## Limitations (v1)

- **Email is synchronous** inside the batch request — no retry queue and no
  background worker. A slow SMTP server slows the request.
- Real-time delivery is Server-Sent Events only, in-memory, and instance-local.
- No cross-instance fanout guarantee.
- No replay stream; clients recover missed events through `GET /notifications`
  and unread-count polling.
- Browser Web Push only; no native mobile push, FCM, APNS, SMS, or paid push
  vendor.
- Push delivery is best-effort and depends on browser/platform permission and
  push-service behavior.
- No per-trip notification preferences.
- No unsubscribe links.
- No quiet hours.
- No email digests.
- `mock` provider is the local-dev default and sends no external mail.
- No WebSockets.
- No RabbitMQ / Kafka / event bus — Trip Service calls this service
  **synchronously over HTTP** after a successful action, fail-open (a
  notification failure never breaks the originating Trip Service action).
- No background workers.

These are intentional and documented so the synchronous HTTP design can be
swapped for an event bus / async worker later without changing the public API.
