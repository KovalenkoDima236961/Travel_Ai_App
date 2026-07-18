# Accepted Security Risks

Exceptions are temporary and reviewed; this file is not a way to silence a
high/critical finding.

| Tool/area | Finding or limitation | Reason and compensating controls | Owner | Review by |
| --- | --- | --- | --- | --- |
| Browser auth | Access/refresh tokens in localStorage | Existing v1 client contract. Short access TTL, rotation, no raw HTML rendering, CSP/reporting and logout cleanup reduce risk. Cookie-session migration needs CSRF and cross-service design. | Platform | 2026-10-18 |
| Web CSP | Report-only policy permits currently required inline/eval directives | Prevents breaking Next local development while violations are measured. `frame-ancestors`, nosniff, referrer, permissions and production HSTS are already emitted. | Web | 2026-10-18 |
| Receipt malware scan | No real scanner adapter is configured | Strict MIME/sniff/size/private-storage/download controls are enforced. `FILE_SCANNING_ENABLED` must remain false until a fail-closed adapter exists. | Trip | 2026-12-18 |
| Public shares | Legacy schema retains raw share token | Required for backward compatibility. Shares can expire/disable and public DTOs are sanitized; migration to a hash-only value is planned. | Trip | 2026-12-18 |
| Semgrep / ESLint | No ESLint-specific browser security rules yet | Semgrep security-audit/OWASP gate scans TypeScript now. Add narrowly reviewed ESLint rules when ESLint is adopted. | Web | 2026-10-18 |
| npm audit | Moderate PostCSS advisory `GHSA-qx2v-qp2m-jg93` through current Next dependency tree | CI blocks high/critical; `npm audit fix --force` proposes a breaking Next downgrade. Review a compatible Next/PostCSS update instead of applying the force fix. | Web | 2026-08-18 |

There are currently no accepted high/critical scanner findings. Any future
entry must include the exact advisory/rule, affected version/path, mitigation,
owner, and expiry.
