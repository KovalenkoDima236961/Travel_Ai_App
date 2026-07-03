# Deployment Checklist

## Pre-deploy

- `APP_ENV` is set to `staging` or `production`.
- `./scripts/validate-env.sh <env-file>` passes.
- Strong secrets are set and no dev defaults remain.
- Database backup completed.
- Migrations tested on staging.
- Images built with `./scripts/build-production-images.sh <env-file>`.
- `CORS_ALLOWED_ORIGINS` matches the public web URL and is not `*`.
- RabbitMQ credentials are not guest defaults.
- Worker concurrency and shutdown timeout are reviewed.
- `OPS_ADMIN_EMAILS` is set if the Ops Dashboard is enabled.
- Metrics, RabbitMQ management, Postgres, and Grafana are not publicly exposed.

## Post-deploy

- Compose services are healthy.
- Web root returns 200.
- Login works.
- Trip creation works.
- Generation job completes.
- Notifications unread count returns.
- Worker consumes jobs and queue depth is normal.
- DLQ is empty or understood.
- Prometheus scrapes services over the internal network.
- Logs show no secret values.
- `./scripts/prod-smoke-test.sh` passes.
