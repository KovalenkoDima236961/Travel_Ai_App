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
- The reverse proxy denies `/internal/*`, `/metrics`, and ops routes.
- JWT, internal-token, public-share, calendar, SMTP, VAPID, and provider secrets
  are distinct and stored outside the repository.
- `AI_PROMPT_LOGGING_ENABLED=false` and `LOG_LLM_PAYLOADS=false`.
- Receipt size/MIME/extension limits are reviewed; scanner fail-open is false.
- Auth/share/receipt rate limits are reviewed for the replica count.
- HTTPS and production HSTS are enabled, and CSP report-only events are monitored.
- `OFFLINE_CACHE_MAX_AGE_DAYS` is set and shared-device guidance is published.

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
- `./scripts/security-smoke-test.sh` passes against a disposable test account.
- Public share output has no expense, receipt, collaborator, activity, policy,
  approval, readiness, calendar, or budget-confidence fields.
- Missing/invalid internal tokens return 401 and receipt downloads return
  `nosniff` plus `private, no-store`.
