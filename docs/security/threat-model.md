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

### AI dataset curation

Training-dataset curation is a separate Trip Service trust boundary. User-derived examples require explicit consent and remain blocked when consent is revoked. Sanitization excludes receipts/OCR, calendar details, comments, private notes, user identifiers, secrets, raw prompts, hidden system instructions, raw logs, and unlicensed provider text before review. Private exports are disabled by default and never served as public URLs.

### AI fine-tuning and adapter runtime

Local fine-tuning is an operator-only workflow behind explicit feature flags.
The training runner consumes curated exports, validates checksums and split
isolation, and never loads holdout examples into training. Adapter runtime
activation is disabled by default, constrained to an approved artifact directory,
and guarded by checksum validation plus environment-specific gates. Runtime
metadata may expose safe keys/checksums/variant names, but not paths, prompts,
responses, user IDs, trip IDs, or provider payloads.

### Offline and service-worker data

Offline keys and pending mutations are user-scoped. Logout removes the active
user's cached private records; permission failures stop retry. The service
worker caches the application shell/immutable assets, not authenticated API,
receipt, or export responses.

### Data lifecycle and deletion capability

Retention cleanup reduces the exposure window for temporary and expired data, but Worker Service has sensitive deletion capability. Its manual cleanup operations require an authenticated ops admin; owning-service cleanup endpoints require `X-Internal-Service-Token`. Cleanup uses bounded ID batches, run locks, dry-run, and aggregate-only logs. Filesystem cleanup resolves paths inside configured storage directories. Audit/security logs are retained unless an explicit policy later enables their cleanup.

### AI model serving

Shadow inference handles sensitive planning context. RabbitMQ messages must
carry safe references and immutable metadata only, not raw prompts, complete
itineraries, calendar details, comments, receipts, profile identity fields,
private addresses, provider secrets, or adapter paths.

Model assignment is a backend decision. Users and public share viewers cannot
submit arbitrary `deploymentKey`, adapter IDs, model paths, rollout salts, or
traffic modes. Frontend flags are UX hints only.

Adapter artifacts are supply-chain sensitive. Candidate serving requires
checksum verification, artifact path confinement, approved adapter status, and
guardrail configuration. Critical guardrail failures must be able to pause
candidate traffic without redeploying.
