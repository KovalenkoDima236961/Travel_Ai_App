# Playbook: add a database migration

1. Confirm the table belongs to this service. Cross-service data needs an API, not a foreign write.
2. Add the next ordered `NNNNNN_description.up.sql` and a matching `.down.sql` only when reverse execution is safe. Follow the service's existing filenames.
3. Make forward SQL transactional where the migration tool/database allows it. Add indexes for new filter/sort/join paths, with a production-safe plan for costly index creation.
4. Update repository DTO/query code and run any sqlc generation already used by that service; do not hand-edit generated output.
5. Apply locally with `./scripts/run-migrations.sh <service>` and inspect `./scripts/migration-status.sh`.
6. Test fresh migration execution through `./scripts/test-backend-integration.sh` and add repository tests for constraints/defaults.
7. For staging/production: take and verify a backup, review locking/data-backfill risk, deploy migration once, verify readiness, and prefer a compensating forward migration over ad-hoc down SQL.

See [data ownership](../../architecture/data-ownership.md) and the [failed-migration runbook](../../operations/runbooks/migrations-failed.md).
# Retention check

For a growing table, document a retention category, safe bounded cleanup query, and whether the record is protected from automatic deletion.
