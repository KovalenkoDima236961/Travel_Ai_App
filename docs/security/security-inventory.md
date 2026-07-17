# Security Inventory

## Data classes

| Class | Examples | Boundary and handling |
| --- | --- | --- |
| `public` | Health status, sanitized public itinerary and route | May be returned anonymously; no private IDs or related records. |
| `authenticated_user` | Account/profile, notification preferences | JWT subject only; a user reads or changes their own record. |
| `trip_private` | Private trip, itinerary, route, comments, activity, readiness | Owner/collaborator/workspace access is resolved in Trip Service. |
| `workspace_private` | Membership, roles, policy, approval, workspace budgets | Workspace role plus resource policy; never inferred from authentication alone. |
| `financial_private` | Budgets, expenses, settlements, budget confidence | Accepted private trip viewers only; never in public-share DTOs. |
| `receipt_private` | Receipt image/PDF, OCR text, storage key | Private file store; authenticated receipt permission; no public URL. |
| `calendar_private` | OAuth tokens, free/busy blocks, event details | Tokens encrypted; only date-level availability summary enters Trip Service. |
| `ai_context_private` | Prompt, itinerary context, preferences, policy summary | Redacted before forwarding/logging; no raw body logging. |
| `internal_service` | Internal lookup, notification batch, reminder processing | Private network plus `X-Internal-Service-Token`. |
| `admin_only` | Ops dashboards/actions, provider quotas, stale job recovery | User JWT and configured email allowlist; optional internal ops token where present. |
| `secret` | JWT/internal/share signing secrets, OAuth/API/SMTP/VAPID keys | Server environment or secret store only; never logged or sent to browsers. |

Sensitive fields include email, phone, exact home address, collaborator identity,
expense notes, receipt OCR, storage paths, event title/location/attendees, push
tokens, refresh/access/share tokens, passwords, and provider credentials.

## Entry points and auth

- Anonymous: health endpoints and `/public/trips/{shareToken}*`. Public share
  status/data still enforce enabled and expiry; protected content also requires a
  short-lived public-share access token.
- Browser APIs: Auth register/login/refresh plus JWT-protected User, Trip,
  Notification, Calendar, availability, and provider routes. CORS is an explicit
  origin allowlist and wildcard origins fail strict startup.
- Internal: Auth user lookup/batch, User workspace access, Notification create
  batch, Trip due-reminder processing, and External Integration price/transport/
  internal calendar operations. All mounted internal groups require the internal
  token middleware.
- Ops: JWT plus `OPS_ADMIN_EMAILS`; the public reverse proxy must not expose ops
  or metrics routes.

## Receipt flow

The multipart handler limits request bytes before parsing. Trip Service checks
declared size, non-empty content, detected bytes, declared MIME consistency, and
extension. Local storage ignores the original path and writes a UUID key under a
private `0700` root with `0600` files. File reads resolve only DB storage keys
and verify the resolved path remains under the root. Downloads re-check trip and
receipt permission and use `nosniff` plus `private, no-store`. Soft delete is
followed by physical deletion; a deletion failure is a safe warning for orphan
cleanup.

`FileScanner` is an optional boundary. The supplied no-op implementation keeps
development dependency-free. Do not set `FILE_SCANNING_ENABLED=true` until a
real scanner is wired; enabled scanning fails closed unless an explicitly
non-production fail-open policy is selected.

## AI flow

Trip Service serializes the typed request, recursively removes disallowed fields,
redacts email/phone/bearer/API-key-like strings, and forwards only sanitized
JSON. Receipt OCR, raw calendar details, credentials, share secrets, file paths,
and user/workspace IDs are removed. Safe preferences, budget totals, itinerary,
route, and aggregate availability remain.

AI Planning logs method/status/timing, not request bodies. Optional local prompt
logging is both redacted and truncated. RAG chunks are placed under a clearly
untrusted context block and suspicious instruction phrases are removed/flagged.

## Offline flow

Authenticated trip stores use `private:{userId}:{tripId}` keys. Public-share
responses never enter those stores. Logout removes the current user's cached
trips, companion snapshots, pending mutations, receipt blobs, settings, metadata,
and sync logs. Receipt blobs require a per-save confirmation and have a delete
control. Startup removes cached trips older than
`OFFLINE_CACHE_MAX_AGE_DAYS` (30 by default). A different user's queue is never
selected, and permission failures require review/discard rather than retry.

The service worker caches only the offline shell and immutable Next assets; it
does not cache API responses.

## Key handling and known follow-ups

Provider keys are read by External Integrations only. Calendar tokens use AES-GCM
with `CALENDAR_TOKEN_ENCRYPTION_KEY`. Raw public share tokens are still stored
in the current schema for compatibility; a future migration should store only a
token hash and reveal the raw value once. The v1 rate limiters are per-process.
Browser JWT storage should move from localStorage to secure httpOnly cookies in a
separately scoped auth migration.

## AI trace records

`ai_generation_traces` is an ops-only, retained diagnostic record. It stores
redacted summaries and controlled errors, never raw prompts, receipt OCR,
calendar details, private comments, credentials, share tokens, or raw RAG
chunks. Optional snapshots require explicit configuration, redaction, and an
ops audit event on access.
