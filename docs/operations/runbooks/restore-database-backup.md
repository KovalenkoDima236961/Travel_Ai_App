# Runbook: restore a database backup

`restore-postgres.sh` intentionally restores local/development databases only
and requires `--yes`. A restore overwrites target data.

1. Identify and verify the intended backup:

   ```bash
   ./scripts/verify-backup.sh <backup-file-or-directory>
   ```

   Local backup output is supplied by `./scripts/backup-postgres.sh --output
   ./backups --gzip`; keep backup location/access controlled.
2. Stop application writers, confirm target environment and database names, and
   create a fresh pre-restore backup if data matters.
3. Restore locally only:

   ```bash
   ./scripts/restore-postgres.sh <backup-file-or-directory> --yes
   ./scripts/migration-status.sh
   ```

4. Start/wait for services, verify readiness and a safe smoke path. Check
   migration status after restore; a backup predating migrations may need the
   approved migration runner before applications write.
5. Staging/production restores require an approved incident/change procedure,
   a verified backup, maintenance/write control, explicit target confirmation,
   and post-restore authorization/data integrity checks. Do not use the local
   script as a shortcut for production.
