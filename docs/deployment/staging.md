# Staging Deployment

Staging should mirror production validation with `APP_ENV=staging`.

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
