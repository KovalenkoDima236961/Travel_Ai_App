# Worker Service

Go worker for long-running Trip Service generation jobs. It consumes RabbitMQ
messages, loads the full job row from the Trip Service database, reuses Trip
Service generation/business logic, and writes the final job, itinerary,
proposal, version, activity, and notification effects.

The Web App does not call Worker Service directly. It creates and polls jobs
through Trip Service.

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

For budget optimization, a completed job means a pending proposal was stored for
review. It does not mean the itinerary changed.

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
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | Broker URL. |
| `GENERATION_JOBS_PREFETCH` | `1` | AMQP prefetch count. |
| `GENERATION_JOBS_MAX_ATTEMPTS` | `3` | Retry limit before DLQ. |
| `GENERATION_JOBS_RETRY_DELAY_SECONDS` | `10` | Retry queue delay. |
| `GENERATION_JOB_MAX_RUNNING_SECONDS` | `600` | Job timeout and stale-running cutoff. |
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

Worker metrics include consumed/acked/nacked/retried/dead-lettered messages,
active jobs, job starts/completions/failures, job duration, and queue delay.

## Development Checks

```bash
make fmt
make vet
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
