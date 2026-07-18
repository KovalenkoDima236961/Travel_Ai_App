# Security Audit & Hardening v1

## Audit metadata

- **Audit date:** 2026-07-18
- **Scope:** Auth, User, Trip, Notification, Worker, External Integrations, AI
  Planning, Web App, Docker Compose, scripts, dependencies, and CI.
- **Method:** architecture and route review, configuration review, focused
  static-sink searches, existing authorization/privacy regression tests, and
  the security tooling documented in [tools.md](tools.md).

## Threat model summary

The primary risks are account/session takeover, IDOR across trip/workspace
objects, collaborator privilege escalation, disclosure from a bearer public
link, unsafe receipt/export handling, untrusted internal callers, secrets or
private context reaching AI/logs, and private browser state on a shared device.
The full model is in [threat-model.md](threat-model.md).

## Services reviewed

| Area | Evidence reviewed |
| --- | --- |
| Auth | `services/auth-service/internal/application/service`, routes, config, refresh repository |
| Authorization/sharing/files | `services/trip-service/internal/security`, routes, receipt/export/share services and tests |
| Internal trust | all `/internal/*` route groups and token middleware |
| Calendar/AI | integration crypto/calendar code; Trip and AI privacy/redaction code and tests |
| Browser | `apps/web/next.config.ts`, token/offline/service-worker code, external-link rendering |
| Infrastructure | Dockerfiles, `.dockerignore`, `infra/docker-compose.yml`, env validation and CI |

## Findings

| Severity | Finding | Affected area / evidence | Status |
| --- | --- | --- | --- |
| High | A simultaneous refresh of one live token could mint multiple successor tokens because revocation updated by ID without requiring `revoked_at IS NULL`. | `services/auth-service/internal/repository/postgres/auth.go` | **Fixed**: atomic conditional revocation now makes the losing refresh invalid. |
| Medium | Directly exposed Auth Service instances could use spoofed forwarding headers as per-IP rate-limit keys after `RealIP` middleware. | `services/auth-service/internal/httpserver/routes.go`, `handler/rate_limit.go` | **Fixed**: capture the transport peer before `RealIP` and use it for credential rate limits. |
| Medium | Token verification accepted any HMAC JWT method rather than the sole issued method. | `services/auth-service/internal/application/service/tokens.go` | **Fixed**: verification now accepts only `HS256`, with regression coverage for `HS512`. |
| Medium | Browser access and refresh tokens remain in `localStorage`; XSS could expose them. | `apps/web/src/shared/api/auth/token-storage.ts` | Accepted risk; CSP/reporting and no raw HTML rendering reduce exposure. Migrate to httpOnly cookie sessions in a separately designed change. |
| Medium | CSP is report-only and currently permits inline styles/scripts and development evaluation. | `apps/web/next.config.ts` | Deferred: collect violations and promote a compatible policy to enforcement. See [accepted-risks.md](accepted-risks.md). |
| Low | Several `target="_blank"` links relied on `noreferrer` implicitly providing opener isolation. | Trip map/itinerary and transport components in `apps/web/src` | **Fixed**: all reviewed new-tab links now explicitly use `rel="noopener noreferrer"`. |
| Medium | Public-share token values are retained in the compatibility schema. | `services/trip-service/migrations/000004_create_trip_shares_table.up.sql` | Deferred: migrate to a token hash, reveal the bearer token only once, and invalidate existing shares in a planned release. |
| Medium | File scanning is an optional no-op boundary in v1. | `services/trip-service/internal/receipts` | Accepted risk; type sniffing, private storage, limits, and authorization remain enforced. Enable a real scanner fail-closed before claiming malware scanning. |
| Medium | `pytest` below 9.0.3 has a local temporary-directory vulnerability (`PYSEC-2026-1845`, CVSS 6.8). | `services/ai-planning-service/requirements-dev.txt` | **Fixed**: development constraint now requires `pytest>=9.0.3,<10.0`. |
| Low | The web app has no ESLint configuration for browser-specific risky patterns. | `apps/web/package.json` | Deferred: Semgrep is the initial code-pattern gate; add reviewed ESLint security rules when the project adopts ESLint rather than adding a second noisy linter. |
| Informational | Compose publishes documented local development ports. | `infra/docker-compose.yml` | Accepted for local profiles only; production deployment must publish web/reverse-proxy ports, not direct service/ops/metrics ports. |

## Controls verified

- Passwords use bcrypt, refresh values are SHA-256 stored and rotated, auth
  endpoints are rate-limited, and strict environments reject weak JWT/internal
  secrets and wildcard CORS.
- The Trip Service evaluates server-side permissions for owner, collaborator,
  workspace, public-share, and ops principals. Existing route/service tests
  exercise pending/removed/random-user and public-share denial paths.
- Public shares use cryptographic tokens, bcrypt-protected unlocks, expiry and
  disable checks, a sanitized DTO, and unlock/read limits.
- Internal route groups require a constant-time `X-Internal-Service-Token`
  comparison. Calendar OAuth values use AES-GCM and availability omits event
  details. AI context strips OCR, calendar details, secrets, share values and
  private comments before forwarding.
- Receipt storage uses generated keys outside a public root, byte sniffing,
  MIME/extension/size validation, re-authorization on download, and no-store
  response headers. Export handling is owner-scoped and expiry-aware.

## Initial scan record

| Tool | Result | Follow-up |
| --- | --- | --- |
| Bandit (high/high) | No high findings; two medium findings are reported below the gate. | Continue in CI. |
| pip-audit runtime | No known vulnerabilities. | Continue in CI. |
| pip-audit development | Initially found `PYSEC-2026-1845` in pytest 8.4.2. | Fixed by the updated 9.0.3 minimum; final audit is clean. |
| npm audit | No high/critical advisories. Two moderate PostCSS advisories are reported through the current Next dependency tree. | Reported below the CI threshold and tracked as a time-bounded exception; the offered force fix is a breaking Next downgrade. |
| gosec / govulncheck | Not installed in this local workspace. | Scripts and CI install/run both across all Go modules. |
| Trivy / Gitleaks / Semgrep | Tooling added but not installed locally. | CI supplies the gate; run `./scripts/security-scan.sh` after local tool installation. |
| ZAP baseline | Not run: it requires a running local stack. | Run `./scripts/security-scan.sh --zap` before release candidates. |

## Follow-up tasks

1. Re-check all scan results on every merge and record only time-bounded
   exceptions in [accepted-risks.md](accepted-risks.md).
2. Design the cookie-session migration and CSP enforcement rollout.
3. Hash public-share tokens in a backward-compatible migration.
4. Add a production malware-scanner adapter before enabling file scanning.
5. Add an authenticated ZAP context/script after test-user bootstrap is stable.
