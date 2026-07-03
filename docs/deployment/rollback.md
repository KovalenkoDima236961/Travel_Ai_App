# Rollback

## Application Rollback

1. Pause or scale workers down first when job processing is unstable.
2. Start the previous image tag with the same env file.
3. Verify service health.
4. Run the smoke test.
5. Resume workers and watch queue depth/DLQ.

## Migration Limits

Migrations are run forward. Down migrations are not part of automated rollback
because data loss risk varies by migration. If a migration is irreversible or
breaks the previous app version, restore Postgres from the pre-deploy backup.

## Database Restore

Set `APP_ENV=production` and require an explicit confirmation:

```sh
CONFIRM_PRODUCTION_RESTORE=restore ./scripts/restore-postgres.sh /path/to/backup.dump
```

After restore, run health checks and the smoke test.

## RabbitMQ and DLQ

RabbitMQ messages are not the system of record. Generation jobs live in
Postgres. If RabbitMQ is lost, queued DB jobs may need a future redispatch tool.
Inspect DLQ messages in the Ops Dashboard before requeueing or discarding.

## Worker Notes

Workers stop consuming on SIGTERM and wait for current work up to
`WORKER_SHUTDOWN_TIMEOUT_SECONDS`. Avoid killing the container unless the job is
known to be stuck. Use stale job recovery thresholds before retrying old running
jobs.
