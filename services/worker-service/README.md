# Worker Service

Dedicated Go worker for long-running Trip Service generation jobs.

It consumes small RabbitMQ messages from `trip.generation.jobs`, loads the full
job row from Postgres, reuses Trip Service generation/business logic, and updates
the existing job, itinerary, proposal, version, activity, and notification tables.
The Web App keeps using the existing Trip Service job creation and polling APIs.

## Job Types

- `full_generation`
- `day_regeneration`
- `item_regeneration`
- `quality_improvement_day`
- `quality_improvement_item`
- `budget_optimization_day`

## Queue Topology

- Exchange: `trip.jobs.exchange` direct, durable
- Main queue: `trip.generation.jobs`, durable, routing key `trip.generation`
- Retry queue: `trip.generation.retry`, durable, routing key `trip.generation.retry`
- Dead-letter exchange: `trip.jobs.dlx` direct, durable
- Dead-letter queue: `trip.generation.dead_letter`, durable, routing key `trip.generation.dead`

The retry queue uses message TTL and dead-letters back to the main exchange.

## Local Run

From the repository root:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

RabbitMQ management UI is available at `http://localhost:15672` with local
credentials `guest` / `guest`.

To run only the worker from source, set the same Postgres, RabbitMQ, and
downstream service env vars used in `infra/.env.example`, then:

```bash
cd services/worker-service
make run
```

## Important Env Vars

- `WORKER_ENABLED=true`
- `WORKER_HTTP_ADDR=:8090`
- `WORKER_SHUTDOWN_TIMEOUT_SECONDS=30`
- `RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/`
- `GENERATION_JOBS_PREFETCH=1`
- `GENERATION_JOBS_MAX_ATTEMPTS=3`
- `GENERATION_JOBS_RETRY_DELAY_SECONDS=10`
- `GENERATION_JOB_MAX_RUNNING_SECONDS=600`

The worker also reads Trip Service env vars for Postgres, AI Planning Service,
External Integrations Service, Notification Service, enrichment, and budget
conversion.

## Health

- `GET /health` returns process liveness.
- `GET /ready` checks Postgres and RabbitMQ consumer/publisher readiness.
- `GET /metrics` exposes Prometheus metrics.

## Retry And Idempotency

Each message contains only `messageId`, `jobId`, `tripId`, `jobType`,
`createdAt`, `requestId`, and `correlationId`. The job ID is the idempotency
key. Completed, failed, cancelled, and already-running duplicate messages are
acknowledged and skipped.

The worker also reads `x-request-id`, `x-correlation-id`, `x-message-type`,
`x-source-service`, and `x-attempts` headers. Request and correlation IDs are
placed into context, logged, and propagated to downstream internal HTTP calls.

Retryable processing failures reset the running job row back to `queued`,
publish a new message to the retry queue with `x-attempts + 1`, and only then
ACK the original message. Terminal failures are persisted to the job row before
the original message is NACKed into the DLQ.

## Limitations

- No transactional outbox yet; Trip Service marks the job failed if queue
  publish fails in fail-closed mode.
- No distributed tracing backend yet. Metrics are exposed locally through
  Prometheus/Grafana.
- The worker writes Trip Service-owned tables directly as the v1 architecture.
- Queue mode requires RabbitMQ; `in_process` mode remains available in Trip
  Service for local fallback and tests.

## Observability

Worker metrics include `worker_messages_consumed_total`,
`worker_messages_acked_total`, `worker_messages_nacked_total`,
`worker_messages_retried_total`, `worker_messages_dead_lettered_total`,
`worker_active_jobs`, `worker_jobs_started_total`,
`worker_jobs_completed_total`, `worker_jobs_failed_total`,
`worker_job_duration_seconds`, and `worker_job_queue_delay_seconds`.

Job logs include `jobId`, `tripId`, `jobType`, `messageId`, `attempt`,
`durationMs`, `errorCode`, `requestId`, and `correlationId` when available.
Panic recovery records `errorCode="panic"`, logs the stack safely, and attempts
to mark the job failed without crashing the worker process.
