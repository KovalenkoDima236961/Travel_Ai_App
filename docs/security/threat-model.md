# Threat Model

## Assets

Account credentials, JWT and refresh tokens, profile data, private trips and
locations, workspace memberships, public-share secrets, expenses/settlements,
receipt files/OCR, exports, calendar OAuth tokens/free-busy data, provider
credentials, push/email credentials, offline browser data, AI prompts/context,
and internal-service tokens are protected assets.

## Actors

| Actor | Trust level and intended access |
| --- | --- |
| Anonymous user | Health routes and sanitized active public shares only. |
| Authenticated user | Own profile and resources granted by Trip/Workspace policy. |
| Trip owner/editor/viewer | Server-evaluated role permissions; editors do not manage shares/collaborators. |
| Workspace owner/admin/member/viewer | Workspace access plus resource policy; no access to unrelated personal trips. |
| Public share viewer | Dedicated, read-only sanitized DTO after share state/unlock checks. |
| Ops admin | Authenticated user in `OPS_ADMIN_EMAILS`; never inferred from a header or share token. |
| Internal service | Private-network caller with an internal service token, scoped by endpoint. |
| Malicious collaborator | Has only their current role; attempts IDOR, role escalation, and content injection. |
| Stolen-token/link attacker | Can use the bearer secret until expiry/revocation; limited by TTL, rotation and rate limits. |
| Compromised browser/device | May expose local browser storage/offline data; logout purge and short tokens reduce persistence. |

## Trust boundaries and data flows

```text
Browser --JWT/CORS--> public API services --internal token--> other services
Browser --share token--> sanitized public Trip endpoints
Trip Service --redacted typed JSON--> AI Planning / local RAG
External Integrations --encrypted tokens--> PostgreSQL / calendar provider
Trip/User services --authorized files--> private receipt/export storage
Worker --internal token/RabbitMQ--> Trip and Notification services
```

- The browser is untrusted: validation and every object permission check happen
  server-side. CORS is an origin policy, not authorization.
- Public-sharing and browser JWTs are separate credentials and cannot cross
  into each other's route groups.
- Internal endpoints require `X-Internal-Service-Token` even on the private
  Compose network; the token is compared in constant time and is never logged.
- Provider APIs and calendar OAuth endpoints are external trust boundaries.

## Security models

### Authentication

Auth Service issues short-lived HS256 access JWTs and opaque, random refresh
tokens. Password hashes use bcrypt; refresh token database values are hashes.
Rotation uses an atomic `revoked_at IS NULL` transition. Strict environments
reject development/default secrets and short keys. Browser storage is currently
localStorage, which is a documented v1 limitation.

### Authorization

Trip Service centralizes permission evaluation across ownership, accepted
collaboration, workspace membership/role, public-share state, and ops role.
Handlers must check permissions before retrieving/mutating an object and
services retain creator/assignee checks for own-only operations. UUIDs are not
authorization.

### Public shares

Shares are bearer secrets with crypto-random tokens, optional bcrypt password,
short public unlock credential, expiry/disable enforcement and per-IP/share
limits. Public responses use a distinct DTO: no receipts, expenses,
collaborators, activity, private budgets, raw tokens, or private diagnostics.

### Files, exports, and backups

Receipt uploads are authorized, size/type/sniff checked, generated under a
private root, and stored without client path names. Downloads re-authorize and
use safe disposition/type plus `private, no-store`. Exports are user-scoped,
expiry-checked and must sanitize archive entry names; receipts are opt-in.
Backups/exports/receipts are ignored by Git and Docker build contexts.

### Calendar and AI privacy

Calendar OAuth tokens are AES-GCM encrypted at rest; status/free-busy output
does not expose titles, descriptions, attendees or raw token values. Trip
Service redacts AI-bound context, excluding OCR, comments, raw calendar data,
credentials, share values and storage paths. AI Planning treats RAG/user text as
untrusted and does not log raw prompts by default.

### Offline and service-worker data

Offline keys and pending mutations are user-scoped. Logout removes the active
user's cached private records; permission failures stop retry. The service
worker caches the application shell/immutable assets, not authenticated API,
receipt, or export responses.
