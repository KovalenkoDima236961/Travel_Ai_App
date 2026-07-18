# Backups And Restore

## Scope

Postgres holds service-owned account, trip, notification, provider, and job
state. `backup-postgres.sh` backs up all five service databases by default:
Auth, User, Trip, Notification, and External Integrations. It never prints a
database password.

Backups do not include:

- RabbitMQ queues (they are delivery triggers, not the system of record).
- Prometheus/Grafana state.
- Environment files or secrets.
- Private receipt and export volumes. Local Compose stores these in
  `receipt_storage`, `trip_export_storage`, and `user_export_storage`; copy
  them separately when their contents must be retained.

## Create And Verify

From a configured local environment:

```sh
./scripts/backup-postgres.sh
./scripts/verify-backup.sh backups/postgres-YYYYMMDDTHHMMSSZ
```

The default command writes one timestamped custom-format dump per database and
a `SHA256SUMS` manifest below `backups/`. Set `BACKUP_DIR` or use `--output` to
store backups elsewhere. Use `--gzip` only when a plain SQL archive is needed:

```sh
BACKUP_DIR=/secure/backup/location ./scripts/backup-postgres.sh
./scripts/backup-postgres.sh --gzip --output /secure/backup/location trip_service
```

For a stronger local drill, restore a single custom dump into a newly-created
temporary database. This requires explicit confirmation and removes only that
temporary database afterwards:

```sh
./scripts/verify-backup.sh backups/postgres-YYYYMMDDTHHMMSSZ/trip_service.dump --restore-test --yes
```

## Restore Locally

Restore is deliberately restricted to `APP_ENV=local`, `development`, or
`test`, and it always requires `--yes` because it replaces existing tables.

```sh
# Restore every database in a timestamped backup directory
./scripts/restore-postgres.sh backups/postgres-YYYYMMDDTHHMMSSZ --yes

# Restore a legacy single trip-service dump
./scripts/restore-postgres.sh backups/trip_service.dump --yes
```

After restoring, run:

```sh
./scripts/migration-status.sh
./scripts/wait-for-ready.sh core
./scripts/smoke-test.sh --core
```

## Migration And Reset Policy

Create and verify a backup before a staging/production migration or a local
volume reset. Production restores are a controlled operations procedure: use a
tested copy in staging or your platform's approved recovery runbook rather than
the local restore helper. `./scripts/dev-reset.sh --yes` destroys named local
volumes; use `--backup` to request a Postgres backup first.

## Schedule

Take daily staging/production backups, retain them according to the data
retention policy, encrypt them at rest, and practice restoration in staging.
Backups must be stored outside the Compose host to survive host loss.
