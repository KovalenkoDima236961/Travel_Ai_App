# API contract changelog

## Unreleased

- **Trip service:** Added safe public feature-flag projection, ops flag
  management/audit routes, and the `feature_disabled` error code. Additive.
- **Release metadata:** Every API service now exposes public, non-sensitive `GET /version` metadata. This is additive and does not change existing endpoint bodies.

Release rule: when an OpenAPI document changes, update this file, regenerate the Web App types, and include the changed specifications in the release artifact.

## 2026-07-18 — v1 foundation

- **All priority services:** Added backend-owned OpenAPI documents and
  generated Web App types. This is non-breaking because existing response
  bodies and paths are preserved.
- **Trip service:** Distinguished `Trip` (private) from `PublicTripResponse`;
  public sharing excludes private user, collaboration, finance and operational
  data. Non-breaking documentation/typing clarification.
- **Error handling:** Added a normalized Web App error adapter that supports
  both legacy string errors and the documented structured envelope.
