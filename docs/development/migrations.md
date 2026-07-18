# Migrations

Each database-owning Go service keeps versioned `up` migrations in its own
`migrations/` directory. The one-shot `migration-runner` applies them in this
order: Auth, User, Trip, Notification, External Integrations.

```bash
# All local migrations (uses infra/.env)
./scripts/run-migrations.sh

# One service
./scripts/run-migrations.sh trip-service

# Inspect applied versions and dirty/pending states
./scripts/migration-status.sh
```

Add a migration using the existing numbered naming convention, with a forward
`*_up.sql` file and a tested `*_down.sql` only when the rollback is safe. Do
not put data resets in normal migrations. Before staging or production changes:

1. Create and verify a backup.
2. Apply the migration runner once.
3. Check `migration-status.sh` and readiness.
4. Deploy application images.

Rollback is a release decision, not an automated `down` command: restore from
a verified backup or deploy a compensating forward migration when data has been
written under the new schema.
