# Observability

Production Observability v1 is a lightweight local Prometheus/Grafana stack for
the Docker Compose environment. It is intended for development and smoke-test
verification, not as a public production monitoring surface.

## Local URLs

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001
- Grafana local credentials: `admin` / `admin`
- RabbitMQ Prometheus metrics: http://localhost:15692/metrics
- RabbitMQ management UI: http://localhost:15672 (`guest` / `guest`)

Service metrics endpoints:

- Auth Service: http://localhost:8082/metrics
- Trip Service: http://localhost:8080/metrics
- User Service: http://localhost:8083/metrics
- External Integrations Service: http://localhost:8084/metrics
- Notification Service: http://localhost:8086/metrics
- AI Planning Service: http://localhost:8000/metrics
- Worker Service: http://localhost:8090/metrics

Start everything with:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

## Dashboards

Grafana provisions these dashboards from
`infra/observability/grafana/dashboards`:

- `API Overview`: HTTP request rate, error rate, p95 latency, top error routes,
  and in-flight requests.
- `Worker Jobs`: job start/complete/failure rates, active jobs, queue delay,
  job duration, retries, and failure codes.
- `External Providers`: provider request/error rate, p95 latency, fallback use,
  cache hit ratio, and recent provider failures.
- `RabbitMQ Overview`: queue depth, ready/unacked messages, publish/consume
  rates, connections/channels, and DLQ depth.

## Correlation IDs

All Go HTTP services and the AI Planning Service understand:

- `X-Request-ID`: one inbound HTTP request.
- `X-Correlation-ID`: the broader workflow across services, jobs, and messages.

If `X-Request-ID` is missing, services generate a UUID. If
`X-Correlation-ID` is missing, it defaults to the request ID. Responses echo
both headers. Internal HTTP clients propagate both headers to other first-party
services. RabbitMQ generation-job messages also carry `requestId`,
`correlationId`, and matching AMQP headers.

Use these IDs in logs to follow a workflow across Trip Service, RabbitMQ, Worker
Service, AI Planning Service, External Integrations Service, and Notification
Service. Do not add request IDs or correlation IDs as Prometheus labels.

## Metrics Rules

Use bounded, low-cardinality labels only:

- Good labels: `service`, `method`, `route`, `status`, `job_type`,
  `error_code`, `provider`, `operation`, `queue`, `message_type`, `result`.
- Do not label metrics by `userId`, `tripId`, `jobId`, `requestId`,
  `correlationId`, raw path, destination, place name, email, prompt, or raw
  error message.

HTTP metrics use route templates where possible, for example
`/trips/{tripID}/generation-jobs/{jobID}`, not raw UUID paths.

## Adding Metrics

For Go services, add counters/histograms/gauges with
`github.com/prometheus/client_golang/prometheus` and expose them through the
existing `/metrics` endpoint. Prefer small service-local metric files near the
feature being measured.

For AI Planning Service, use `app/observability.py` and keep labels bounded by
operation/result/mode. Never log or label full prompts, user preferences,
private itinerary JSON, tokens, API keys, OAuth codes, or provider error bodies.

## Limitations

- No production alerting rules yet.
- No OpenTelemetry Collector, tracing backend, Loki, Tempo, Jaeger, or service
  mesh in v1.
- `/metrics` is unauthenticated for local Docker networking. Do not expose it
  publicly in production without network controls.
- Grafana credentials are local-development defaults only.
