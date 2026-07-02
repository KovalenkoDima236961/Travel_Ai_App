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

## Retry And Idempotency

Each message contains only `messageId`, `jobId`, `tripId`, `jobType`, and
`createdAt`. The job ID is the idempotency key. Completed, failed, cancelled,
and already-running duplicate messages are acknowledged and skipped.

Retryable processing failures reset the running job row back to `queued`,
publish a new message to the retry queue with `x-attempts + 1`, and only then
ACK the original message. Terminal failures are persisted to the job row before
the original message is NACKed into the DLQ.

## Limitations

- No transactional outbox yet; Trip Service marks the job failed if queue
  publish fails in fail-closed mode.
- No distributed tracing or metrics stack.
- The worker writes Trip Service-owned tables directly as the v1 architecture.
- Queue mode requires RabbitMQ; `in_process` mode remains available in Trip
  Service for local fallback and tests.
