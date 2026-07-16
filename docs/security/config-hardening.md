# Security Configuration Hardening

## Required strict-environment posture

For `APP_ENV=staging` or `production`:

- use unique JWT, internal-service, public-share, database, RabbitMQ, calendar,
  provider, SMTP, VAPID, and ops secrets;
- keep JWT/internal/share secrets at least 32 characters and database passwords
  at least 16;
- set an explicit HTTPS CORS origin; `*` and production localhost are rejected;
- keep `AUTH_REQUIRED=true`;
- keep `AI_PROMPT_LOGGING_ENABLED=false` and legacy
  `LOG_LLM_PAYLOADS=false`; strict AI startup rejects either;
- set `CALENDAR_TOKEN_ENCRYPTION_KEY` to exactly 16, 24, or 32 bytes when
  calendar is enabled;
- leave `FILE_SCANNING_FAIL_OPEN=false` whenever scanning is enabled;
- configure `OPS_ADMIN_EMAILS` before enabling ops.

## Internal token rotation

Receivers prefer comma-separated `INTERNAL_SERVICE_TOKENS` when it is non-empty
and otherwise use `INTERNAL_SERVICE_TOKEN`. Rotate without downtime:

1. Set `INTERNAL_SERVICE_TOKENS=old,new` on every receiver and deploy.
2. Set `INTERNAL_SERVICE_TOKEN=new` on callers and deploy.
3. Confirm `internal_auth_failures_total` is stable.
4. Set receiver list to only `new`, then clear the plural variable after all
   configs use the new singular value.

Strict config validates every value in the plural list. Never put tokens in
command lines, logs, issue text, or browser variables.

## Limits

Defaults are login/register 10/min, refresh 30/min, share unlock 5/min,
public-share reads 120/min, and receipt upload 20/min. These fixed-window
limiters are local to one process. Keep one replica or add a shared Redis-backed
limiter before relying on aggregate multi-instance enforcement.

Receipt defaults are 10 MiB and JPEG/PNG/WebP/PDF with matching extensions.
Scanner disabled is an accepted v1 limitation, not proof a file is malware-free.

## Web

CSP begins in report-only mode because Next currently requires inline styles and
development evaluation. Collect violations, remove unsafe directives, then
promote to enforcing CSP. HSTS is emitted only from a production Next process
and must be used behind HTTPS. Tokens remain in localStorage; never add
`dangerouslySetInnerHTML` for user content without a reviewed sanitizer.

The client cache retention value is built as
`NEXT_PUBLIC_OFFLINE_CACHE_MAX_AGE_DAYS`; Compose maps
`OFFLINE_CACHE_MAX_AGE_DAYS` to it.

