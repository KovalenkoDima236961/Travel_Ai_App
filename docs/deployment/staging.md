# Staging Deployment

Staging should mirror production validation with `APP_ENV=staging`.

## Performance configuration

Apply Trip Service migration `000034` and Notification Service migration `000004` before deploying the matching application images. Size `POSTGRES_MIN_CONNS`/`POSTGRES_MAX_CONNS` from measured concurrency, then start with `DB_QUERY_TIMEOUT_SECONDS=10` and `DB_SLOW_QUERY_THRESHOLD_MS=250`. The compact summary defaults to an enabled 30-second, 1000-entry cache with an 8-second section deadline; tune only after reviewing cache and DB dashboards.

Keep the worker retry/DLQ/stale-running settings explicit in the staging environment and confirm the retry/DLQ queues are declared before traffic. After rollout, run both performance smoke scripts against a seeded trip and inspect the Performance & Reliability dashboard for API p95, DB p95/pool saturation, cache ratio, provider cooldowns, and worker retries.

## Setup

1. Copy `infra/.env.staging.example` to `infra/.env.staging`.
2. Use staging hostnames in `PUBLIC_WEB_BASE_URL` and `NEXT_PUBLIC_*`.
3. Use strong, unique staging secrets.
4. Validate:

```sh
./scripts/validate-env.sh infra/.env.staging
```

## Providers

Staging may use mock providers:

- `PLACE_PROVIDER=mock`
- `ROUTE_PROVIDER=mock`
- `WEATHER_PROVIDER=mock`
- `EMAIL_PROVIDER=mock`
- `WEB_PUSH_ENABLED=false`
- `CALENDAR_PROVIDER=mock`

When a real provider is selected, startup validation requires the matching key
or OAuth secret.

## Notification digests

Apply Notification Service migration `000005` before deploying the matching
service image. Keep at least one Worker Service instance running with
`NOTIFICATION_DIGEST_WORKER_ENABLED=true`; multiple workers are safe because
due batches are claimed atomically. Exercise quiet-hours boundaries, trip mutes,
and one digest retry in staging, then verify the delivery-decision and digest
created/sent/failed metrics before promotion.

## Deploy

```sh
./scripts/build-production-images.sh infra/.env.staging
TRAVEL_AI_ENV_FILE=.env.staging docker compose --env-file infra/.env.staging -f infra/docker-compose.prod.yml run --rm migration-runner
TRAVEL_AI_ENV_FILE=.env.staging docker compose --env-file infra/.env.staging -f infra/docker-compose.prod.yml up -d
```

Grafana and Prometheus can be exposed in staging only behind VPN, basic auth, or
another protected network. They are not published by default.

## Smoke Test

Run `scripts/prod-smoke-test.sh` against staging after every deployment. Use a
dedicated staging smoke user and mock email/push settings unless the test is
explicitly validating those channels.
