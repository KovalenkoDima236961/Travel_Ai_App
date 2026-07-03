# Backups And Restore

## What To Back Up

- Postgres databases: trips, auth, user profiles, notifications, and external
  integration state.
- Environment and secrets separately from database backups.
- Future uploaded files or object storage, if added.

## What Not To Rely On

- RabbitMQ queues as durable business history.
- Container filesystems.
- Prometheus metrics as a source of application state.

## Backup

Use `DATABASE_URL` or `POSTGRES_*`/`PG*` values:

```sh
DATABASE_URL=postgres://user@host:5432/trip_service ./scripts/backup-postgres.sh backups/trip_service.dump
```

For Docker Compose:

```sh
set -a
. infra/.env.production
set +a
./scripts/backup-postgres.sh backups/trip_service.dump
```

## Restore

Staging:

```sh
./scripts/restore-postgres.sh backups/trip_service.dump
```

Production requires explicit confirmation:

```sh
APP_ENV=production CONFIRM_PRODUCTION_RESTORE=restore ./scripts/restore-postgres.sh backups/trip_service.dump
```

## Verification

1. Start the stack.
2. Check `/health` and `/ready`.
3. Run `./scripts/prod-smoke-test.sh`.
4. Confirm workers are consuming and DLQ is understood.

## Schedule

Run daily backups for staging and production. Test restore regularly in staging.

## RabbitMQ Recovery

Jobs are stored in Postgres and RabbitMQ messages trigger work. If RabbitMQ data
is lost, completed history remains in Postgres. Queued jobs may need manual
retry through the Ops Dashboard or a future redispatch tool. Inspect DLQ before
requeueing messages.
