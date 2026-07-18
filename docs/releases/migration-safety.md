# Migration release safety

## Review checklist

For every migration, identify affected service/database, lock and runtime cost, backfill plan, rollback/forward-fix plan, and the earliest app version that depends on it. Update `CHANGELOG.md` **Migration Notes** whenever a migration ships.

- Prefer additive, indexed, nullable, or defaulted changes first.
- Test up migrations on a fresh database with `./scripts/release/check-migrations.sh ci`.
- When practical, test on a restored copy/staging-like database and measure the migration duration.
- Take and verify a production backup before applying production migrations; see [backup and restore](../deployment/backups.md).
- Never introduce a destructive drop, irreversible data rewrite, or long table lock without a reviewed backup and forward-fix plan.

## Expand/contract pattern

1. **Expand:** add schema that old and new application versions both tolerate.
2. **Migrate/backfill:** deploy compatible application behavior and bounded backfill work.
3. **Contract:** remove old columns/data only in a later release after all supported application images no longer read them.

Release an app that can read both representations before writing only the new one. Make data jobs idempotent and observable.

## Rollback limits

An application image rollback never rolls back the database. Only run a down migration that is explicitly reviewed as safe and does not discard or reinterpret newer data. For most schema/data failures, stop the harmful path, restore only when approved and verified, or ship a forward-compatible fix. Record the exact database version, backup reference, and decision in the incident/change record.
