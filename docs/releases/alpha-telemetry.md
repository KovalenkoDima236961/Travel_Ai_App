# Alpha telemetry, performance, and capacity

Closed-alpha telemetry is intentionally minimal. It is for launch safety and product learning, not user profiling.

## Product and reliability signals

| Signal | Source | Privacy rule |
| --- | --- | --- |
| Registration/profile/trip creation completion | Existing auth/user/trip events or smoke reports | Aggregate counts only |
| Generation request/success/failure | `generation_jobs_*`, AI trace summaries | No prompt or itinerary text |
| Median/p95 generation duration | `ai_request_duration_seconds`, worker job duration | Labels limited to operation/mode/status |
| Queue delay and retry/failure rate | `worker_job_queue_delay_seconds`, worker metrics | No user/trip labels |
| Schema repair rate | `ai_repair_attempts_total`, Trip validation metrics | Operation/result only |
| Fallback rate | `ai_provider_fallbacks_total` | Provider/operation/reason only |
| Grounded-place and quality warning rate | Trip generation quality summaries | Aggregate by status/category |
| Bad-place/usefulness feedback | Personalization feedback summary | Chip type and safe metadata only |
| Itinerary edit/version rate | Trip revision/version counts | Aggregate counts only |
| Public share creation and public read failures | Trip Service share routes | No token values in metrics |
| Export usage | Export job counts | No filenames or content |
| OpenAI token usage | `ai_provider_input_tokens_total`, `ai_provider_output_tokens_total` | Provider/operation/model group only |
| OpenAI cost estimate | Spend checker or ledger metric | Preserve billing currency; UAH is estimated |
| Circuit-breaker events | Provider circuit metrics | Provider/operation only |

## Data that must not be telemetry

Do not send or store raw trip titles, exact destinations in high-cardinality metrics, full itineraries, comments, receipts, OCR text, private notes, emails, user names, tokens, raw prompts, raw AI responses, provider payloads, or share passwords.

## Feedback loop

Post-generation feedback uses existing personalization feedback:

- `entityType=general`
- `entityId=<generation job id>`
- `tripId` and `workspaceId` stay in private authenticated storage
- metadata keys: `source=alpha_generation_feedback`, `category=<quality status>`, `style=[jobType, primary_used|fallback_used]`

Ops should not draw quality conclusions until at least 10 feedback samples exist for a comparable prompt/model/version segment.

## Performance targets

| Path | Alpha target |
| --- | --- |
| Web first useful load | p75 <= 3.0s on invited-user supported devices |
| Trip list | p95 API <= 800ms |
| Trip detail summary | p95 API <= 1200ms |
| Generation job creation | p95 <= 800ms |
| OpenAI generation | median <= 90s, p95 <= 180s |
| Worker queue delay | p95 <= 5m |
| Notifications unread count | p95 <= 500ms |
| Public share load | p95 <= 1000ms |
| API 5xx error rate | < 1% over 30 minutes |

Use existing performance smoke scripts and alpha smoke reports. Do not create a new load-test platform for alpha v1.

## Capacity assumptions

- Invited users: 25-50.
- Daily generations: 50-100.
- Max concurrent OpenAI calls: 3 by default.
- Worker concurrency: 1 by default for safer alpha recovery.
- RabbitMQ queue warning: queue delay p95 > 5 minutes or backlog > 20 generation messages.
- DB pool sizes follow `infra/.env.alpha.example`; raise only after observing saturation.
- Daily UAH spend cap starts at 500 UAH and must be reviewed against provider billing currency.

## Storage and backup expectations

- PostgreSQL backup uses `scripts/backup-postgres.sh` and verification uses `scripts/verify-backup.sh`.
- Receipt files live under `RECEIPT_STORAGE_LOCAL_DIR`; OCR remains disabled by feature flag.
- Export files live under `TRIP_DATA_EXPORT_STORAGE_DIR`/`USER_DATA_EXPORT_STORAGE_DIR` and have a cleanup TTL.
- AI eval reports and provider artifacts must be redacted before artifact upload.
- Alpha rollback scripts do not restore non-DB storage automatically; record file-storage state and manual restore expectations in the launch decision.
