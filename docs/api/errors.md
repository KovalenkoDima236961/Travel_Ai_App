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
| `validation_error` | 400/422 | Bind field errors when available; keep entered safe values. |
| `not_found` | 404 | Show missing/deleted resource state; avoid leaking private existence. |
| `conflict` | 409 | Refetch state and ask user to resolve. |
| `itinerary_conflict` | 409 | Refetch current itinerary/revision; merge or explicitly retry. |
| `edit_lock_conflict` | 409 | Show advisory editor lock/presence and allow a safe later retry. |
| `rate_limited` | 429 | Back off; honor retry guidance. |
| `provider_rate_limited` | 429/503 | Display temporary provider degradation/fallback status. |
| `provider_quota_exceeded` | 429/503 | Use fallback if supplied; do not loop retries. |
| `provider_unavailable` | 502/503 | Preserve local edits and offer retry. |
| `generation_failed` | 422/500/503 | Show sanitized job failure and a guarded retry action. |
| `upload_invalid_type` | 400/415 | Reject file before/after upload; show allowed formats. |
| `upload_too_large` | 413 | Show size limit; do not retry unchanged file. |
| `public_share_expired` | 404/410 | Show expired share; do not reveal private trip data. |
| `public_share_password_required` | 401/403 | Show unlock form; rate-limit attempts. |
| `internal_auth_required` | 401/403 | Service configuration/caller bug; never expose token to browser. |

## Rules for new handlers

Use stable machine-readable codes, neutral user-safe messages, bounded field
details, and request ID propagation. Do not include SQL errors, stack traces,
secret values, authorization headers, provider tokens, raw prompts, or raw OCR
in an API error.

## Related docs

- [API overview](overview.md)
- [Troubleshooting](../development/troubleshooting.md)
- [Security-sensitive feature playbook](../development/playbooks/add-security-sensitive-feature.md)
