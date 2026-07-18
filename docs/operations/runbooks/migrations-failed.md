# Runbook: migrations failed

1. Stop repeated deployment retries and capture the first failing migration:

   ```bash
   ./scripts/migration-status.sh
   docker compose -f infra/docker-compose.yml --env-file infra/.env logs migration-runner
   ```

2. Determine whether failure is SQL syntax, missing dependency, privilege,
   duplicate/manual schema change, lock, or an already-dirty version. Do not
   edit an applied migration or mark a version clean blindly.
3. In local development, fix the forward migration or local configuration and
   rerun `./scripts/run-migrations.sh`. A disposable stack may be reset only
   after `./scripts/backup-postgres.sh --output ./backups --gzip` and explicit
   `./scripts/dev-reset.sh --yes`.
4. In staging/production, back up and verify first. Pause rollout, inspect the
   schema with an approved read-only query, and choose a reviewed compensating
   forward migration or verified restore. Do not run blanket `down` migrations
   after application writes may have occurred.
5. After recovery, rerun the migration runner once, confirm status/readiness,
   smoke the affected service boundary, and record the incident/change.

Related: [migration playbook](../../development/playbooks/add-database-migration.md).
