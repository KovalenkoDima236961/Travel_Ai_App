# API errors

## Current and target error vocabulary

Error serialization is not fully uniform across legacy Go handlers. Treat HTTP
status, handler message, and request ID as authoritative today. New or changed
frontend-facing endpoints should converge on this shape without breaking
existing clients:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid request.",
    "details": [{"field": "startDate", "message": "Start date is required."}],
    "requestId": "..."
  }
}
```

| Code | Typical status | Client behavior |
| --- | ---: | --- |
| `unauthorized` | 401 | Refresh/login through existing auth flow; do not retry unchanged credentials. |
| `forbidden` | 403 | Hide/restrict action and explain missing role; never retry as another user. |
| `feature_disabled` | 403 | Hide the optional control and show the neutral unavailable state; do not retry until an operator enables it. |
| `validation_error` | 400/422 | Bind field errors when available; keep entered safe values. |
| `not_found` | 404 | Show missing/deleted resource state; avoid leaking private existence. |
| `conflict` | 409 | Refetch state and ask user to resolve. |
| `itinerary_conflict` | 409 | Refetch current itinerary/revision; merge or explicitly retry. |
| `edit_lock_conflict` | 409 | Show advisory editor lock/presence and allow a safe later retry. |
| `rate_limited` | 429 | Back off; honor retry guidance. |
| `search_invalid_query` | 400 | Clear control characters from the search box and retry with normal text. |
| `search_query_too_long` | 400 | Trim the query to the documented maximum before retrying. |
| `search_invalid_filter` | 400 | Remove unsupported result types, scope, trip, or workspace filters. |
| `search_rate_limited` | 429 | Debounce/back off global search requests for the current user. |
| `provider_rate_limited` | 429/503 | Display temporary provider degradation/fallback status. |
| `provider_quota_exceeded` | 429/503 | Use fallback if supplied; do not loop retries. |
| `provider_unavailable` | 502/503 | Preserve local edits and offer retry. |
| `generation_failed` | 422/500/503 | Show sanitized job failure and a guarded retry action. |
| `upload_invalid_type` | 400/415 | Reject file before/after upload; show allowed formats. |
| `upload_too_large` | 413 | Show size limit; do not retry unchanged file. |
| `public_share_expired` | 404/410 | Show expired share; do not reveal private trip data. |
| `public_share_password_required` | 401/403 | Show unlock form; rate-limit attempts. |
| `internal_auth_required` | 401/403 | Service configuration/caller bug; never expose token to browser. |
| `ai_dataset_consent_required` | 409 | Do not approve/export the example; ask for explicit user consent or choose non-user-derived examples. |
| `ai_dataset_consent_revoked` | 409 | Treat the example as ineligible; remove it from pending approval/export workflows. |
| `ai_dataset_sanitization_failed` | 409 | Keep the example blocked; inspect sanitizer findings and do not bypass review. |
| `ai_dataset_quality_too_low` | 409 | Keep the example pending/rejected; improve or replace the candidate before retrying approval. |
| `ai_dataset_duplicate` | 409 | Resolve the duplicate group before including the example in a version. |
| `ai_dataset_version_exists` | 409 | Pick a new semantic dataset version or inspect the existing one. |
| `ai_dataset_version_not_ready` | 409 | Wait for/build a ready version before export/download. |
| `ai_dataset_export_disabled` | 503 | Hide export/download actions unless `AI_DATASET_EXPORT_ENABLED=true` is deliberately configured. |
| `ai_dataset_export_failed` | 500 | Keep the version unexported and inspect server logs/storage permissions. |
| `ai_dataset_private_data_detected` | 409 | Block approval/export and route the candidate to privacy review. |
| `ai_dataset_license_not_allowed` | 409 | Exclude provider/copyrighted content that is not licensed for training use. |
| `ai_training_disabled` | 403/503 | Hide experiment controls unless `AI_FINE_TUNING_EXPERIMENTS_ENABLED=true` is deliberately configured. |
| `ai_training_dataset_invalid` | 409/422 | Stop the run; validate export shape, checksums, consent/sanitization, and task distribution. |
| `ai_training_dataset_not_frozen` | 409 | Require a frozen/exported dataset version before any training run. |
| `ai_training_holdout_leak_detected` | 409 | Abort the run and invalidate the candidate; holdout data must never enter training. |
| `ai_training_model_incompatible` | 409 | Choose a Hugging Face-compatible base/revision and record license/lineage before retrying. |
| `ai_training_hardware_unsupported` | 503 | Keep the experiment queued/failed; run on an approved local GPU host or lower the method footprint. |
| `ai_training_failed` | 500 | Keep the candidate unpromoted; inspect local training logs and checkpoint state. |
| `ai_adapter_load_failed` | 503 | Fail readiness unless explicit base fallback is configured; do not serve the candidate blindly. |
| `ai_adapter_checksum_mismatch` | 409/503 | Reject the adapter load/promotion until the recorded checksum and artifact checksum match. |
| `ai_evaluation_failed` | 500 | Keep promotion controls disabled until validation/test reports are regenerated. |
| `ai_promotion_gate_failed` | 409 | Block staging/production approval; show failed gates and require a manual reject/retrain decision. |
| `ai_model_variant_unavailable` | 404/503 | Fall back only when explicitly configured; otherwise mark the requested variant unavailable. |
| `unknown_error` | any | Preserve safe local state and show a generic retryable failure with request ID when available. |

## Rules for new handlers

Use stable machine-readable codes, neutral user-safe messages, bounded field
details, and request ID propagation. Do not include SQL errors, stack traces,
secret values, authorization headers, provider tokens, raw prompts, or raw OCR
in an API error.

## Related docs

- [API overview](overview.md)
- [API conventions](conventions.md)
- [Troubleshooting](../development/troubleshooting.md)
- [Security-sensitive feature playbook](../development/playbooks/add-security-sensitive-feature.md)
