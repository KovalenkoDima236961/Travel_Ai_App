# Security and Privacy

Security Hardening v1 establishes practical, deny-by-default boundaries for the
current application. The primary threats are cross-user trip access, collaborator
privilege escalation, public-share leakage, malicious receipt uploads, exposed
internal endpoints, secret or PII disclosure to AI/logs, and private data left in
a shared browser.

## Controls

- Trip authorization is resolved once and checked through
  `services/trip-service/internal/security`. Unknown principals, roles, and
  permissions are denied.
- Public shares expose a dedicated sanitized DTO. They are read-only, can expire,
  can be disabled immediately, use bcrypt password hashes, and have per-IP/share
  unlock limits.
- Receipts use private server storage, generated storage keys, content sniffing,
  MIME/extension/size checks, authorization on every download, and no-store
  response headers.
- Every `/internal/*` route is behind constant-time service-token validation.
  `INTERNAL_SERVICE_TOKENS` permits overlap during rotation.
- AI-bound JSON is redacted in Trip Service. AI Planning treats retrieved
  documents as untrusted, neutralizes prompt-injection markers, and never logs
  raw prompts by default.
- IndexedDB keys are user-scoped. Logout clears that user's records, stale trip
  copies are purged, receipt drafts require consent, and a 403 stops sync retry.
- The Web App emits anti-sniffing, frame, referrer, permissions, report-only CSP,
  and production HSTS headers.
- Sensitive routes use v1 in-memory rate limits. These counters are
  instance-local and require a shared store before scaling to many replicas.

See [audit.md](audit.md), [threat-model.md](threat-model.md),
[security-inventory.md](security-inventory.md),
[access-control-matrix.md](access-control-matrix.md), and
[config-hardening.md](config-hardening.md). The contribution checklist and CI
gate instructions are in [secure-development-checklist.md](secure-development-checklist.md)
and [tools.md](tools.md).

## Testing

Run the Go, Python, and Web unit suites, then `scripts/security-smoke-test.sh`
against a running local stack. The regression tests cover role boundaries,
public response sanitization/expiry/disable/unlock, rotating internal tokens,
receipt type/path controls, AI redaction and RAG injection handling, rate limits,
and offline user separation.

Run `./scripts/security-scan.sh` for the repeatable SAST, dependency, secret,
filesystem, and configuration scan suite. Append `--zap` after starting the
local stack for the unauthenticated OWASP ZAP baseline.

## User-facing limitations

Public links remain bearer secrets. A share password reduces casual access but
is not a substitute for inviting a private collaborator. Receipts and offline
copies may contain sensitive data and should be removed on shared devices. AI
context is filtered, but users should not put credentials or secrets in trip
text. Access/refresh tokens remain in `localStorage` in v1; CSP reduces but does
not eliminate XSS risk, and migration to secure httpOnly cookies remains future
work.
