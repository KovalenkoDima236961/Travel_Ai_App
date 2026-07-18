# Storage growth

Use the admin-only `GET /ops/storage/summary` and cleanup task/runs endpoints to identify a category without exposing user data. Start every investigation with the matching dry-run cleanup task and retain a backup before any retention reduction.

Check whether the growth is in active receipts, audit data, jobs, export packages, or provider cache. Active receipts and audit data are intentionally protected. For temporary export/OCR files, verify configured base paths and use the documented retention setting; never delete based on a hand-built broad filesystem command.

For local backups, first run:

```sh
BACKUP_DIR=./backups ./scripts/cleanup-backups.sh --dry-run
```

Use `--yes` only after verifying the directory, age threshold, and backup restore posture. Production backups follow the separate operational backup policy.
