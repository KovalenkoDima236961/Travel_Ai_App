# Alpha rollback

Use when an alpha release candidate must be paused, rolled back to prior service images, or forward-fixed.

## First response

1. Stop invitations and announce alpha pause internally.
2. Capture current image tags, Git SHA, service versions, migration version, queue depth, failed jobs, and backup status.
3. Disable the narrowest risky feature flag if it contains impact faster than image rollback.
4. Create and verify a backup before any service image switch.

## Rehearsal command

```bash
./scripts/release/rehearse-alpha-rollback.sh --env-file infra/.env.alpha --previous-image-tag <prior-tag>
```

The script does not run unsafe down migrations and does not restore into the staging alpha DB.

## Decision rules

- Frontend-only issue: redeploy prior web image and verify `/api/version`, login, trip read, and public share read.
- Backend issue without schema incompatibility: redeploy affected service image and run alpha smoke.
- Worker issue: pause/replace Worker only, preserve queue/DLQ, and verify one bounded job.
- AI issue: use feature flags/fallback before image rollback.
- Migration issue: follow `docs/releases/migration-safety.md`; prefer forward fix unless restore is approved.

## Verification

- `scripts/release/check-versions.sh staging`
- `scripts/release/alpha-smoke-test.sh --mock-openai --env-file infra/.env.alpha`
- `scripts/migration-status.sh --env-file infra/.env.alpha`
- Backup verification output.

## Escalate

Escalate when schema compatibility cannot be proven, backup verification fails, public data exposure occurred, or rollback would reintroduce a known security issue.
