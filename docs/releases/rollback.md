# Rollback playbook

Always capture the deployed image SHA, timestamps, errors, queue depth, and database migration version first. Verify the previous image version with `./scripts/release/check-versions.sh --expected-version <version> --expected-sha <sha>` after action. A database is never rolled back automatically.

| Scenario | Symptoms and decision | Rollback / verification | Unsafe rollback / forward fix |
| --- | --- | --- | --- |
| Frontend only | Broken UI, asset load failure, or browser regression with stable APIs. | Redeploy prior `web-app:<version>-<sha>`; check `/api/version`, login, one trip read/write. | Unsafe only when the UI already writes an incompatible API/data shape; publish a compatible UI fix. |
| Backend service | Elevated 5xx, readiness failure, or isolated contract regression. | Stop/redeploy only the affected service to its prior SHA tag; run health, ready, version, and the affected smoke flow. | Do not roll back if a newer migration is required by the old binary; deploy a forward-compatible service fix. |
| Worker | Queue backlog, repeated jobs, or bad side effects. | Pause/scale down worker consumption, preserve queue/DLQ, deploy prior worker image, then verify readiness and a bounded job. | Never replay/delete messages blindly; patch idempotency or queue handling forward. |
| AI service | Generation timeout, invalid output, or bad model integration. | Switch Trip Service to mock/fallback where configured or redeploy prior AI image; run mock generation/repair smoke. | If persisted output is affected, retain it and repair with a forward application change. |
| Migration issue | Failed migration, dirty state, locks, or incompatible schema. | Stop rollout, preserve logs, assess backup and compatibility; follow [migration safety](migration-safety.md). | Down migrations are exceptional. Prefer an additive schema change or restoration approved by the incident owner. |
| API contract mismatch | Web/API parse failures or generated client mismatch. | Roll back the consumer or provider to the last compatible pair; verify OpenAPI and generated client checks. | If other consumers depend on the new contract, add compatibility rather than removing it. |
| Provider integration | Quota/error spike, unexpected external responses, or cost risk. | Disable/switch to mock/fallback configuration where supported; roll back only the integration service if needed. | Avoid retry storms and never expose provider credentials in diagnostics. |
| Security hotfix gone wrong | Authentication failure, over-broad block, or new disclosure risk. | Contain access, rotate affected credentials, and choose prior image only if it is safer; verify auth and audit paths. | Do not restore a known vulnerable image. Ship a minimal forward security fix and document the risk decision. |

After any rollback, run release smoke, monitor errors and queues through the agreed window, update the incident/change record, and start the forward-fix follow-up.
