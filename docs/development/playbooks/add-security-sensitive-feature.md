# Playbook: add a security-sensitive feature

1. Update or reference the relevant [threat model](../../security/threat-model.md) and define assets, actors, abuse cases, and trust boundary.
2. Enforce authorization server-side and add deny/role/isolation tests. Derive identity from auth context; do not trust body user IDs or client roles.
3. Validate and bound every input. For files, validate type/size/content handling/storage authorization; for public data, sanitize DTOs explicitly.
4. Redact logs and telemetry, minimize retained payloads, set rate limits where abuse is realistic, and keep internal endpoints inaccessible to browsers.
5. Add security regression tests and run `./scripts/security-scan.sh`; document any accepted risk in the existing accepted-risks process.
6. Update API, operational, and user-facing documentation in the same change.
