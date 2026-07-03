# Production Deployment

This v1 deployment target is Docker Compose on a VM or a basic container host.
It does not assume Kubernetes, Helm, Terraform, a service mesh, or managed
secrets.

## Architecture

- Public entry points: Web App, Auth Service, User Service, External
  Integrations Service. Trip, Notification, and Worker traffic is normally
  proxied by the Web App, but Trip and Notification ports are bind-mounted to
  localhost by default for smoke tests or a local reverse proxy.
- Internal services: Postgres, RabbitMQ, AI Planning Service, Worker Service,
  Prometheus, and Grafana.
- Metrics are scraped by Prometheus over the Docker network. Do not publish
  `/metrics` publicly.
- Auth uses bearer JWTs. The browser stores tokens client-side; production
  security relies on HTTPS, strong JWT secrets, tight CORS origins, and avoiding
  token logging.

## Environment

1. Copy `infra/.env.production.example` to `infra/.env.production`.
2. Replace every placeholder with a strong value.
3. Validate without printing secrets:

```sh
./scripts/validate-env.sh infra/.env.production
```

`APP_ENV=production` enables strict startup validation in Go, Python, and Web
configuration paths. Public `NEXT_PUBLIC_*` values must be HTTPS and cannot use
localhost.

## Build

```sh
./scripts/build-production-images.sh infra/.env.production
```

The Web App receives `NEXT_PUBLIC_*` as build args because Next.js embeds those
values in browser bundles.

## Migrations

Back up Postgres first, then run the one-shot migration container:

```sh
docker compose --env-file infra/.env.production -f infra/docker-compose.prod.yml run --rm migration-runner
```

Migrations use each service's existing `golang-migrate` startup path through
the `cmd/migrate` binaries. They only run `up`; destructive rollbacks are manual.
`infra/docker-compose.prod.yml` defaults container `env_file` to
`infra/.env.production`; set `TRAVEL_AI_ENV_FILE=.env.staging` or another file
name only when intentionally using a different file.

## Start

```sh
docker compose --env-file infra/.env.production -f infra/docker-compose.prod.yml up -d
```

By default backend API host ports bind to `127.0.0.1` except the Web App. Put a
reverse proxy or firewall in front of them. If you intentionally expose backend
APIs directly, set the `*_SERVICE_BIND` values and keep `CORS_ALLOWED_ORIGINS`
limited to `PUBLIC_WEB_BASE_URL`.

## Public Ports

- Public: Web App `3000`.
- Localhost/reverse-proxy only by default: Auth `8082`, User `8083`, Trip
  `8080`, Notification `8086`, External Integrations `8084`.
- Not public: Postgres, RabbitMQ, RabbitMQ management, Worker, AI Planning,
  Prometheus, Grafana, metrics endpoints.

## Verification

```sh
BASE_WEB_URL=https://app.example.com \
AUTH_SERVICE_URL=https://auth.example.com \
TRIP_SERVICE_URL=https://trip.example.com \
NOTIFICATION_SERVICE_URL=https://notifications.example.com \
TEST_USER_EMAIL=smoke+prod@example.com \
TEST_USER_PASSWORD='set-a-test-password' \
./scripts/prod-smoke-test.sh
```

Use mock providers for smoke tests when real provider quotas or emails should
not be touched.

## Worker Safety

- Deploy one worker for v1 unless the AI/provider backend is proven safe under
  concurrency.
- Keep `WORKER_CONCURRENCY` low and increase only after watching queue depth,
  active jobs, and DLQ.
- On SIGTERM the worker cancels consumption, waits up to
  `WORKER_SHUTDOWN_TIMEOUT_SECONDS`, and only ACKs messages after processing.
- `GENERATION_JOB_MAX_RUNNING_SECONDS` and `OPS_STALE_RUNNING_JOB_SECONDS`
  control stale job recovery. Check the Ops Dashboard before retrying jobs.

## Operational Notes

- RabbitMQ is a trigger queue, not the source of truth. Job state is in Postgres.
- The container filesystem is disposable.
- Prometheus and Grafana are internal by default. Expose them only through a
  protected network, VPN, or reverse proxy authentication.
