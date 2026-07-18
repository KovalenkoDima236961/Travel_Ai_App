# Data Retention & Cleanup Policies v1

This policy reduces unnecessary storage and exposure. It is an engineering policy, not a legal-compliance guarantee. Retention is configurable by environment; production defaults are conservative and scheduled cleanup is observable through Worker Service.

| Category | Owner/location | Default retention and action | Configuration | Safety notes |
| --- | --- | --- | --- | --- |
| Expired / revoked refresh tokens | Auth Service / `refresh_tokens` | Hard-delete 30 days after `expires_at` or `revoked_at` | `RETENTION_EXPIRED_REFRESH_TOKENS_DAYS`, `RETENTION_REVOKED_REFRESH_TOKENS_DAYS` | Active tokens are never selected. Login/rate-limit records are in-memory in v1. |
| Read / unread notifications | Notification Service / `notifications` | Read: 180 days; unread: 365 days; bounded hard-delete | `RETENTION_READ_NOTIFICATIONS_DAYS`, `RETENTION_UNREAD_NOTIFICATIONS_DAYS` | Preferences and current notifications are never targeted. |
| Digests / disabled push subscriptions | Notification Service | Final digest batches: 180 days; disabled subscriptions: 180 days | `RETENTION_NOTIFICATION_DIGESTS_DAYS`, `RETENTION_INACTIVE_PUSH_SUBSCRIPTIONS_DAYS` | Active push subscriptions are preserved. Delivery-attempt retention is documented when a dedicated table is added. |
| Generation jobs | Trip Service / `trip_generation_jobs` | Completed, failed and cancelled: 90 days; queued/running jobs are retained | `RETENTION_COMPLETED_JOBS_DAYS`, `RETENTION_FAILED_JOBS_DAYS` | Stale running jobs are marked failed under the existing timeout policy, not deleted. |
| Proposals / discovery / route alternatives | Trip Service | Pending proposals become expired; expired/discarded/failed proposals: 180 days | proposal retention config (v1 default 180 days) | Applied records retain a compact user-visible history. |
| Activity / audit events | Trip Service | Activity: 365 days when enabled; audit: no deletion by default | `RETENTION_ACTIVITY_EVENTS_DAYS`, `RETENTION_AUDIT_EVENTS_DAYS`, `RETENTION_AUDIT_CLEANUP_ENABLED=false` | Audit/security events are never silently hard-deleted. |
| Soft-deleted comments | Trip Service | Hard-delete after 180 days, or redact to a tombstone where counts/history require it | `RETENTION_SOFT_DELETED_COMMENTS_DAYS` | Only comments already deleted by a user are eligible. |
| Public-share sessions | Trip Service | Unlock sessions: token TTL + 7 days; expired access sessions: 7 days | `RETENTION_PUBLIC_SHARE_SESSIONS_DAYS` | Active shares and public trip data remain until explicitly disabled/deleted. |
| Exports | Trip Service / private export storage | Temporary and failed files: 7 days; metadata: 90 days | `RETENTION_EXPORT_FILES_DAYS`, `RETENTION_EXPORT_METADATA_DAYS` | Keys are server-generated; cleanup resolves paths within the configured export directory and skips in-progress files. |
| Receipts / OCR | Trip Service / receipts and `receipt_ocr_results` | Receipt files follow their expense; raw OCR: 30 days after review; temp uploads: 24 hours; orphan files: 7 days | `RETENTION_RAW_OCR_RESULTS_DAYS`, `RETENTION_TEMP_UPLOAD_FILES_HOURS`, `RETENTION_ORPHANED_RECEIPT_FILES_DAYS` | Never delete a receipt referenced by an active expense. |
| AI traces | Trip Service / redacted trace tables | 30 days (existing trace configuration may be stricter) | `AI_OBSERVABILITY_RETENTION_DAYS` | Raw prompts are not a supported storage mode; no prompts/responses are written to cleanup logs. |
| Presence/edit locks | Trip Service | In-memory only; existing TTL cleanup applies | existing lock TTL config | No database cleanup is needed in v1. |
| Offline client data | Browser IndexedDB and service-worker caches | Cached trips: optional stale cleanup (default 30 days); pending mutations preserved; drafts require confirmation | `NEXT_PUBLIC_OFFLINE_CACHE_MAX_AGE_DAYS` | User-scoped clear/logout cleanup is available. |
| Provider caches / OAuth state | External Integrations Service | Persisted cache: expiry + 7 days; OAuth state: 1 day | `RETENTION_PROVIDER_CACHE_GRACE_DAYS`, `RETENTION_OAUTH_STATES_DAYS` | Current v1 provider caches are in-memory and self-evict. Disconnection removes encrypted calendar tokens. |
| Provider quota counters | External Integrations Service | 400 days | provider quota retention config | Retain long enough for operational reporting. |
| Worker retry/DLQ metadata | RabbitMQ / Worker Service | Broker policy; do not delete active retry/DLQ messages | broker operational policy | Cleanup run records provide aggregate history, not payload retention. |
| Backups | Local backup directory | Keep local files 30 days by default | `RETENTION_LOCAL_BACKUPS_DAYS` | Worker never deletes production backups. Use `scripts/cleanup-backups.sh --dry-run` first. |

## Execution model and safeguards

Worker Service is the orchestrator. It calls `POST /internal/cleanup/{taskName}` on the service that owns the data, with `X-Internal-Service-Token`; it does not write into another service's tables. Each operation accepts `dryRun`, `batchSize`, and `maxBatches`, uses ID-limited batches, and is idempotent: missing rows/files are skipped.

The scheduled v1 task registry contains only owning-service handlers that are present in this release (Auth, Notification, and External Integrations). The Trip Service rules above deliberately remain policy-only until its existing jobs, receipts, export-file, and audit routes can be migrated as one protected release; they are not registered as failing no-op tasks.

`cleanup_runs` records aggregate counts, status, request ID, warnings, and a bounded lock expiry. A partial unique index prevents concurrent runs of the same task. No run record or structured log includes token values, receipt filenames, OCR text, raw prompts, personal data, or file contents. Metrics use task/status labels only.

Scheduled cleanup is enabled with `CLEANUP_JOBS_ENABLED=true` and `WORKER_SCHEDULED_JOBS_ENABLED=true`; v1 accepts a daily UTC cron in the form `minute hour * * *`. `CLEANUP_FAIL_OPEN=false` stops the remaining tasks in that scheduled pass after a failure; `true` records the failure and continues independent tasks. Local and test default to dry-run. Destructive mode requires explicitly setting `CLEANUP_DRY_RUN_DEFAULT=false`; unknown environments fail configuration validation.

## Non-goals

- No automatic account deletion.
- No legal or regulatory compliance guarantee.
- No cloud-provider or paid-object-store lifecycle policy.
- No automatic deletion of audit/security logs unless a later explicit policy enables it.
- No production-backup deletion by the application worker.
