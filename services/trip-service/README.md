# Trip Service

Go service that owns trip planning state and the main domain workflow for the
Travel AI App. It stores trips, itineraries, revision history, collaborators,
comments, public shares, activity events, generation jobs, budget proposals,
calendar sync mappings, accommodation data, and enrichment metadata.

Trip Service is the orchestration point between user-facing APIs, AI generation,
external provider data, notifications, and background workers.

## Architecture

```mermaid
flowchart TD
    Client["Web App / API client"] --> HTTP["http-server<br/>chi router + middleware"]
    HTTP --> Handler["handlers<br/>decode, validate, map errors"]
    Handler --> Service["application service<br/>business rules"]

    Service --> Repo["trip repository<br/>squirrel + pgx"]
    Repo --> DB[("PostgreSQL<br/>trip_service")]

    Service --> Generator["ItineraryGenerator port"]
    Generator --> Mock["Mock generator"]
    Generator --> AI["AI Planning Service<br/>HTTP mode"]

    Service --> User["User Service<br/>profile/preferences"]
    Service --> External["External Integrations<br/>places, weather, prices, rates, calendar"]
    Service --> Notify["Notification Service<br/>internal batch"]
    Service --> Rabbit["RabbitMQ publisher<br/>queue dispatch mode"]

    Worker["Worker Service"] --> Rabbit
    Worker --> DB
    Worker --> AI
    Worker --> External
    Worker --> Notify
```

The service follows a layered structure: `http-server` -> `application` ->
`domain`, with adapters in `infrastructure`. The composition root in
`internal/app` wires config, logger, Postgres, providers, HTTP server, workers,
and graceful shutdown.

## Responsibilities

| Area | Owned by Trip Service |
| ---- | --------------------- |
| Trip access | Owner/collaborator access checks, role capabilities, public share redaction. |
| Itinerary safety | `itineraryRevision`, revision-aware writes, version snapshots, restore. |
| Generation | Job creation, sync compatibility routes, AI context assembly, result validation. |
| Collaboration | Invites, roles, accepted/shared trips, presence, soft edit locks. |
| Workspaces | Personal vs workspace trips, workspace role checks via User Service, combined effective access. |
| Activity | Persistent audit feed plus in-memory SSE best-effort updates. |
| Comments | Private item comments, counts, edit/delete permissions. |
| Budget | Trip budget, item/accommodation costs, multi-currency summaries, analytics, proposals. |
| Accommodation | One private structured stay per trip, included in AI/budget/route context. |
| Sharing | One public read-only link per trip, optional expiry/password unlock. |
| Calendar | Per-trip/user sync state; provider operations delegated to External Integrations. |

## Revision-Safe Writes

```mermaid
sequenceDiagram
    participant W as Web App
    participant T as Trip Service
    participant DB as Postgres

    W->>T: GET /trips/{id}
    T-->>W: itineraryRevision = 12
    W->>T: PUT /trips/{id}/itinerary expectedItineraryRevision=12
    T->>DB: UPDATE ... WHERE id=? AND itinerary_revision=12
    alt row updated
        T->>DB: Insert itinerary version and activity event
        T-->>W: itineraryRevision = 13
    else stale revision
        T-->>W: 409 itinerary_conflict, currentItineraryRevision
    end
```

Every private itinerary-changing request must include
`expectedItineraryRevision`. Stale writes fail with `409 itinerary_conflict`.
Comments, collaborators, shares, presence, notifications, budget settings,
accommodation settings, exports, and public views do not increment the itinerary
revision.

## Background Jobs

```mermaid
stateDiagram-v2
    [*] --> queued
    queued --> running: worker claims row
    running --> completed: result saved
    running --> failed: terminal error
    running --> queued: retryable error
    queued --> cancelled: user cancels
    failed --> [*]
    completed --> [*]
    cancelled --> [*]
```

Generation job types:

- `full_generation`
- `day_regeneration`
- `item_regeneration`
- `quality_improvement_day`
- `quality_improvement_item`
- `budget_optimization_day`

Dispatch modes:

- `GENERATION_JOB_DISPATCH_MODE=queue`: publish a small RabbitMQ message and let
  Worker Service process the existing DB job row.
- `GENERATION_JOB_DISPATCH_MODE=in_process`: use the Trip Service local poller
  for fallback and tests.

Queue messages intentionally contain only IDs, type, timestamps, and
correlation metadata. They do not contain access tokens, prompts, preferences,
or itinerary JSON.

## Endpoint Groups

| Group | Routes |
| ----- | ------ |
| Health | `GET /health`, `GET /ready`, `GET /metrics` |
| Trips | `POST /trips`, `GET /trips`, `GET /trips/shared-with-me`, `GET /trips/{id}` |
| Generation jobs | `POST /trips/{id}/generation-jobs`, `GET /trips/{id}/generation-jobs`, `GET /trips/{id}/generation-jobs/{jobId}`, `POST /trips/{id}/generation-jobs/{jobId}/cancel` |
| Sync generation compatibility | `POST /trips/{id}/generate`, day regeneration, item regeneration |
| Itinerary | `PUT /trips/{id}/itinerary`, version list/detail/restore routes |
| Budget | `GET /trips/{id}/budget-summary`, `PUT /trips/{id}/budget`, budget optimization job/proposal routes |
| Cost analytics | `GET /trips/{id}/analytics/costs`, `GET /workspaces/{workspaceId}/analytics/costs` |
| Accommodation | `GET /trips/{id}/accommodation`, `PUT /trips/{id}/accommodation`, `DELETE /trips/{id}/accommodation` |
| Collaboration | collaborator CRUD/accept/decline, `GET /collaboration/invitations` |
| Presence and locks | `/trips/{id}/presence*`, `/trips/{id}/edit-lock` |
| Comments | `/trips/{id}/comments`, `/trips/{id}/comments/counts`, comment update/delete |
| Activity | `GET /trips/{id}/activity`, `GET /trips/{id}/activity/stream` |
| Sharing | `GET/POST/PATCH/DELETE /trips/{id}/share`, public share status/unlock/read routes |
| Calendar | `GET/POST/DELETE /trips/{id}/calendar-sync/google*` |

Private routes require `Authorization: Bearer <accessToken>` when
`AUTH_REQUIRED=true`. Public share routes use opaque share tokens and optional
short-lived public share unlock tokens.

## Workspace Trips

Trips now have nullable `workspace_id`. Existing rows remain personal trips with
`workspace_id=NULL`; workspace trips keep `user_id` as creator/audit owner while
access is granted through User Service workspace roles.

`POST /trips` accepts optional `workspaceId`. If present, Trip Service calls
User Service `POST /internal/workspaces/access-check` and requires workspace
`owner`, `admin`, or `member`; `viewer` can view but cannot create/edit.

`GET /trips` accepts:

- `scope=all|personal|workspace`
- `workspaceId=<uuid>` for a single workspace

For workspace listings, Trip Service calls
`POST /internal/workspaces/list-for-user` and returns only trips from active
memberships. Trip responses include `workspaceId`, `scope`, and access metadata
with `source=owner|workspace|collaborator|public`.

Effective access is the strongest safe permission from personal owner,
workspace role, direct trip collaborator, or public share. Workspace owner/admin
map to owner-level trip management, member maps to editor, and viewer maps to
viewer. Direct trip collaborators still work for workspace trips, including
non-workspace exceptions. Public share links remain separate anonymous read-only
access and never expose workspace member data.

## Cost Analytics

Cost Analytics Dashboard v1 is read-only and computed from existing Trip Service
data at request time. It does not add accounting records or booking/payment
data.

- `GET /trips/{id}/analytics/costs?currency=EUR` returns trip-level estimated
  totals, budget remaining/overage, cost by day/category/source/confidence,
  original currency totals, expensive items, missing/uncertain estimate counts,
  conversion warnings, and actionable planning insights.
- `GET /workspaces/{workspaceId}/analytics/costs?currency=EUR&from=2026-01-01&to=2026-12-31`
  aggregates accessible workspace trips by trip/category/source/month and
  includes top trips/items plus incomplete budget warnings.
- Trip analytics requires private trip access. Owners, editors, and viewers can
  read analytics; public share tokens do not expose analytics in v1.
- Workspace analytics requires an active workspace role through User Service.
  Owner, admin, member, and viewer roles can read the dashboard.

Calculations reuse the budget conversion rules used by `budget-summary`.
Accommodation cost is included once in total/category rollups and not forced
into daily totals. Currency conversion failures are returned as warnings and the
affected costs remain visible in original-currency totals.

Limitations: costs are estimates for planning only; exchange rates may be
approximate; provider prices and availability may change; missing estimates can
make totals incomplete; reports are not accounting, tax, invoice, payment, or
financial-advice features.

## Important Configuration

| Variable | Purpose |
| -------- | ------- |
| `HTTP_ADDRESS`, `HTTP_WRITE_TIMEOUT` | HTTP bind address and long generation response timeout. |
| `AUTH_REQUIRED`, `JWT_ACCESS_SECRET`, `AUTH_HEADER_NAME` | Auth Service JWT validation. |
| `ITINERARY_GENERATOR_MODE` | `mock` or `http` AI generator adapter. |
| `AI_PLANNING_SERVICE_URL`, `AI_PLANNING_TIMEOUT_SECONDS` | AI Planning Service client. |
| `USER_SERVICE_URL`, `USER_CONTEXT_*` | Profile/preference lookup for personalization. |
| `WORKSPACES_ENABLED`, `USER_SERVICE_URL`, `WORKSPACE_ACCESS_TIMEOUT_SECONDS`, `INTERNAL_SERVICE_TOKEN` | Workspace access checks and trip list scoping. |
| `EXTERNAL_INTEGRATIONS_SERVICE_URL` | Weather, places, prices, rates, and calendar calls. |
| `WEATHER_CONTEXT_*` | Optional weather context for AI prompts. |
| `PLACE_ENRICHMENT_*`, `PRICE_ENRICHMENT_*` | Auto-enrichment after generation. |
| `BUDGET_CONVERSION_*` | Exchange-rate conversion for budget summaries. |
| `PUBLIC_SHARING_*`, `PUBLIC_SHARE_ACCESS_*` | Public share link controls. |
| `TRIP_PRESENCE_*`, `TRIP_ACTIVITY_STREAM_*`, `TRIP_EDIT_LOCK_*` | In-memory SSE/advisory collaboration features. |
| `GENERATION_JOB_*`, `RABBITMQ_*` | Job queue, retry, DLQ, and worker behavior. |
| `OPS_DASHBOARD_ENABLED`, `OPS_ADMIN_EMAILS`, `OPS_STALE_RUNNING_JOB_SECONDS` | Allowlisted admin job monitor and safe job actions. |
| `NOTIFICATIONS_*`, `NOTIFICATION_SERVICE_*` | Synchronous fail-open notification fanout. |
| `CALENDAR_SYNC_*`, `DEFAULT_CALENDAR_TIMEZONE` | Calendar sync behavior. |
| `POSTGRES_*`, `POSTGRES_MIG_PATH` | Database and auto-migration settings. |

## Ops Dashboard Endpoints

When `OPS_DASHBOARD_ENABLED=true`, allowlisted users can inspect generation jobs
with `GET /ops/jobs`, `GET /ops/jobs/summary`, and `GET /ops/jobs/{jobId}`.
Safe mutations require a non-empty `reason`: retry creates a new queued job,
cancel only affects queued jobs, and mark-failed only affects stale running jobs.

See [configs/config.example.yaml](configs/config.example.yaml) and
[.env.example](.env.example) for the full local template.

## Run Locally

From this service directory:

```bash
cp .env.example .env
set -a; source .env; set +a
make run
```

Run with YAML config:

```bash
cp configs/config.example.yaml configs/config.yaml
make config-run
```

Run the full application stack from the repository root:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

Migrations run automatically on startup. Manual migration commands:

```bash
make migrate-up
make migrate-down
```

## Development Checks

```bash
make fmt
make vet
make test
make build
```

## Operational Notes

- `queue` dispatch requires RabbitMQ and Worker Service. Keep `in_process` as a
  local fallback only.
- Notification calls are synchronous but fail-open by default; a notification
  outage must not break the originating trip action.
- Weather, place, price, and budget conversion provider calls are fail-open by
  default in local development and produce warnings or partial context.
- Presence, activity SSE, and edit locks are process-local v1 features. They do
  not provide cross-instance guarantees.
- Public share responses are sanitized and omit private collaborator,
  notification, activity, version-management, accommodation, and budget proposal
  surfaces.

## Observability And Safety

- `GET /metrics` exposes HTTP, job, notification, activity, provider, and domain
  metrics.
- Logs and internal calls propagate `X-Request-ID` and `X-Correlation-ID`.
- Do not log access tokens, internal service tokens, share passwords, public
  share access tokens, full prompts, full preference payloads, full private
  itinerary JSON, OAuth tokens, or provider API keys.
