# Secure Development Checklist

Use this in every change that adds or changes an endpoint, job, data flow, UI
rendering path, integration, or storage path.

- [ ] Is authentication required, and is the route mounted behind the right middleware?
- [ ] Is authorization checked server-side for the exact object and action?
- [ ] Does ownership, collaborator status, workspace role, and archived state behave correctly?
- [ ] Is a public-share route separate and limited to a sanitized public DTO?
- [ ] Does every `/internal/*` route require `X-Internal-Service-Token`?
- [ ] Are inputs bounded and validated (IDs, filters, sort fields, dates, body size, MIME)?
- [ ] Are outputs redacted for tokens, passwords, OCR, calendar data, raw paths, and private fields?
- [ ] Is an appropriate rate limit applied to credential, upload, generation, export, or public unlock flow?
- [ ] Are filenames, storage keys, ZIP entries, redirects, and external URLs safe and allowlisted?
- [ ] Do logs redact token/secret/password/key/credential/authorization/cookie fields?
- [ ] Are metrics low-cardinality and free of PII, prompt text, tokens, and raw URLs?
- [ ] Does AI context exclude secrets, OCR, private comments, and calendar details, and label user text untrusted?
- [ ] Does offline state remain user-scoped and get removed on logout/account switch?
- [ ] Are authorization, IDOR, public-share, and error-path regression tests included?
- [ ] Are config, docs, accepted risks, and security-tool baselines updated?
