# Worker Service

Go worker for long-running Trip Service generation jobs. It consumes RabbitMQ
messages, loads the full job row from the Trip Service database, reuses Trip
Service generation/business logic, and writes the final job, itinerary,
proposal, version, activity, and notification effects.

The Web App does not call Worker Service directly. It creates and polls jobs
through Trip Service.

Worker Service also runs the Smart Pre-Trip Reminders polling loop. That loop
does not read reminder rows directly; it calls Trip Service
`POST /internal/reminders/process-due` with the shared internal service token so
Trip Service remains the owner of reminder permissions, notification recipient
selection, and sent/failed state transitions.

## Go Package Layout

- `cmd/worker` is a thin executable entrypoint.
- `internal/app` is the composition root and lifecycle owner.
- `internal/httpserver` contains health, readiness, metrics, and ops HTTP routes.
- `internal/rabbitmq` contains worker-specific queue consumption, retry/DLQ
  behavior, generation-job RabbitMQ ops wrappers, and worker metrics.
- `internal/config` loads worker runtime and management settings while reusing
  Trip Service generation config.
- `pkg` contains worker-local project-agnostic plumbing such as logging,
  shutdown coordination, HTTP/request observability, and RabbitMQ
  connection/management API helpers. Do not put job processing or Trip Service
  business logic there.

## Processing Flow

```mermaid
sequenceDiagram
    participant T as Trip Service
    participant Q as RabbitMQ
    participant W as Worker Service
    participant DB as trip_service DB
    participant AI as AI Planning Service
    participant E as External Integrations
    participant N as Notification Service

    T->>DB: Insert queued job row
    T->>Q: Publish small job message
    W->>Q: Consume message
    W->>DB: Claim job by jobId
    W->>E: Optional weather/place/price/rate context
    W->>AI: Generate/regenerate/optimize
    W->>DB: Save revision-aware result
    W->>N: Best-effort notification
    W->>Q: ACK original message
```

Messages contain only routing metadata: `messageId`, `jobId`, `tripId`,
`jobType`, timestamps, request ID, and correlation ID. The database job row is
the source of truth.

## Reminder Worker Flow

```mermaid
sequenceDiagram
    participant W as Worker Service
    participant T as Trip Service
    participant N as Notification Service

    W->>T: POST /internal/reminders/process-due
    T->>T: Find due pending reminders
    T->>N: POST /internal/notifications/batch
    T-->>W: processed/sent/failed counts
```

The reminder worker runs on a configurable interval, defaults to every five
minutes, and sends a bounded `limit` per poll. It is intentionally coarse:
sub-minute precision, SMS/WhatsApp delivery, calendar reminder export, and
recurring reminders are outside v1. Trip Service handles idempotency by skipping
already sent/completed/disabled/deleted reminders.

## Queue Topology

```mermaid
flowchart LR
    Exchange["trip.jobs.exchange<br/>direct durable"] --> Main["trip.generation.jobs"]
    Main --> Worker["worker consumer"]
    Worker -->|retryable failure| Retry["trip.generation.retry<br/>TTL delay"]
    Retry --> Exchange
    Worker -->|terminal failure / max attempts| DLX["trip.jobs.dlx"]
    DLX --> DLQ["trip.generation.dead_letter"]
    Worker -->|success / duplicate terminal job| Ack["ACK"]
```

Default topology:

| Name | Default |
| ---- | ------- |
| Exchange | `trip.jobs.exchange` |
| Main queue | `trip.generation.jobs` |
| Main routing key | `trip.generation` |
| Retry queue | `trip.generation.retry` |
| Retry routing key | `trip.generation.retry` |
| Dead-letter exchange | `trip.jobs.dlx` |
| Dead-letter queue | `trip.generation.dead_letter` |
| Dead-letter routing key | `trip.generation.dead` |

## Job Types

- `full_generation`
- `day_regeneration`
- `item_regeneration`
- `quality_improvement_day`
- `quality_improvement_item`
- `budget_optimization_day`
- `policy_repair`

For budget optimization, a completed job means a pending proposal was stored for
review. It does not mean the itinerary changed.
For policy repair, a completed job means a pending repair proposal was stored
for review. The worker never applies the repaired itinerary directly.

Jobs that persist or apply itinerary output run through Trip Service AI
Generation Reliability validation. When a version is saved, the worker copies
the latest itinerary version `metadata.generationQuality` into the completed
job `resultPayload.generationQuality`, and Trip Service exposes the same object
as a direct `generationQuality` field on job responses.

## Idempotency And Retries

```mermaid
stateDiagram-v2
    [*] --> received
    received --> skipped: job already terminal
    received --> running: claim queued job
    running --> completed: save succeeds
    running --> retry: retryable error
    retry --> received: delayed message
    running --> failed: terminal error or max attempts
    skipped --> [*]
    completed --> [*]
    failed --> dlq
    dlq --> [*]
```

The `jobId` is the idempotency key. Completed, failed, cancelled, and duplicate
already-running jobs are acknowledged and skipped. Retryable failures reset the
job row to `queued`, publish a delayed retry message, and then ACK the original
message. Terminal failures are persisted before the message is NACKed into the
DLQ.

## Provider Rate-Limit / Quota Errors

When a generation job calls External Integrations Service and a provider is
limited, the service returns a controlled error. Trip Service clients surface
these as typed `providerlimit.Error`s, and `generationjobs.ClassifyJobError`
classifies them (ahead of the generic dependency-failure branch):

| Provider error | Job error code | Classification | Behaviour |
| -------------- | -------------- | -------------- | --------- |
| `provider_rate_limited` | `provider_rate_limited` | transient | Retried via the delayed retry queue while attempts remain, then DLQ'd. |
| `provider_limits_unavailable` | `provider_limits_unavailable` | transient | Retried like a rate limit (used when fail-open is off and the quota store is down). |
| `provider_quota_exceeded` | `provider_quota_exceeded` | terminal | Failed immediately (no retry) so the worker does not tight-loop against an exhausted daily quota. Ops can retry the job the next day. |

Retries use the existing TTL-delayed retry queue (bounded by
`GENERATION_JOBS_MAX_ATTEMPTS`), so provider-limited jobs are never hammered in a
tight loop. Persisted job error codes and messages are safe — they name the
provider category (e.g. "Provider rate limited: route_estimate") but never expose
API keys, account details, or quota internals.

Note: these codes only surface for job steps that require the provider (i.e. the
relevant enrichment/context step is configured fail-closed). Steps left
fail-open continue without enrichment on a limit and the job still completes.

## Local Development

Run the full stack from the repository root:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

RabbitMQ management UI:

```text
http://localhost:15672
guest / guest
```

Run only the worker from source after exporting the same Postgres, RabbitMQ, AI,
external integration, notification, enrichment, and budget-conversion variables
used by `infra/.env.example`:

```bash
cd services/worker-service
make run
```

## Important Configuration

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `WORKER_ENABLED` | `true` | Enable processing loop. |
| `WORKER_HTTP_ADDR` | `:8090` | Health/ready/metrics server. |
| `WORKER_CONCURRENCY` | `1` | Local processing concurrency. |
| `WORKER_SHUTDOWN_TIMEOUT_SECONDS` | `30` | Graceful shutdown window. |
| `REMINDER_WORKER_ENABLED` | `true` | Enable due reminder polling. |
| `TRIP_SERVICE_URL` | `http://trip-service:8080` | Trip Service internal endpoint base URL. |
| `REMINDER_WORKER_POLL_INTERVAL_SECONDS` | `300` | Reminder polling interval. |
| `REMINDER_WORKER_BATCH_SIZE` | `100` | Maximum reminders to process per poll. |
| `REMINDER_WORKER_LOOKAHEAD_MINUTES` | `0` | Optional due-processing lookahead. |
| `REMINDER_WORKER_TIMEOUT_SECONDS` | `10` | HTTP timeout for Trip Service reminder processing. |
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | Broker URL. |
| `RABBITMQ_MANAGEMENT_URL` | `http://rabbitmq:15672` | RabbitMQ Management API for queue/DLQ ops. |
| `RABBITMQ_MANAGEMENT_USER`, `RABBITMQ_MANAGEMENT_PASSWORD` | `guest` | Management API credentials. |
| `GENERATION_JOBS_PREFETCH` | `1` | AMQP prefetch count. |
| `GENERATION_JOBS_MAX_ATTEMPTS` | `3` | Retry limit before DLQ. |
| `GENERATION_JOBS_RETRY_DELAY_SECONDS` | `10` | Retry queue delay. |
| `GENERATION_JOB_MAX_RUNNING_SECONDS` | `600` | Job timeout and stale-running cutoff. |
| `OPS_DASHBOARD_ENABLED`, `OPS_ADMIN_EMAILS` | disabled, empty | Enables allowlisted worker ops routes. |
| `AI_PLANNING_SERVICE_URL` | compose service URL | Downstream AI call target. |
| `EXTERNAL_INTEGRATIONS_SERVICE_URL` | compose service URL | Weather/place/price/rate target. |
| `NOTIFICATION_SERVICE_URL` | compose service URL | Internal notification fanout. |
| `POSTGRES_*` | local compose defaults | Trip Service database access. |

Worker Service also reads many Trip Service configuration variables because it
executes the same generation and enrichment logic.

## Health And Metrics

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/health` | Process liveness. |
| `GET` | `/ready` | Postgres and RabbitMQ readiness. |
| `GET` | `/metrics` | Prometheus metrics. |
| `GET` | `/ops/worker/status` | Worker readiness and active jobs. |
| `GET` | `/ops/queues/status` | Main/retry/DLQ queue depths. |
| `GET` | `/ops/dlq/messages` | Sanitized DLQ message headers. |
| `POST` | `/ops/dlq/messages/{messageId}/requeue` | Requeue one DLQ message with a reason. |
| `POST` | `/ops/dlq/messages/{messageId}/discard` | Discard one DLQ message with a reason. |

Worker metrics include consumed/acked/nacked/retried/dead-lettered messages,
active jobs, job starts/completions/failures, job duration, and queue delay.

## Development Checks

```bash
make fmt
make vet
make lint
make test
make build
```

## Limitations

- No transactional outbox yet; Trip Service can fail a job immediately when
  publish fails in fail-closed mode.
- No distributed tracing backend yet; metrics and correlation IDs are the local
  observability path.
- Worker writes Trip Service-owned tables directly in v1.
- Queue mode requires RabbitMQ. Trip Service `in_process` dispatch remains the
  local fallback and test path.

## Safety

- Never put access tokens, prompts, preferences, or itinerary JSON in RabbitMQ
  messages.
- Logs include job IDs, trip IDs, job type, attempt, duration, request ID, and
  correlation ID, but must not include tokens, full prompts, full preference
  payloads, or provider secrets.

## AI trace propagation

Generation messages preserve job, request, and correlation IDs. Trip Service's
generation worker creates or updates its safe AI trace from those IDs, records
high-level stage events, and marks the trace completed or failed with a
controlled code. Trace persistence is fail-open and must never change AMQP
ack/retry or DLQ behavior.
