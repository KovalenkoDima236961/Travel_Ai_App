# API contract changelog

## 2026-07-18 — v1 foundation

- **All priority services:** Added backend-owned OpenAPI documents and
  generated Web App types. This is non-breaking because existing response
  bodies and paths are preserved.
- **Trip service:** Distinguished `Trip` (private) from `PublicTripResponse`;
  public sharing excludes private user, collaboration, finance and operational
  data. Non-breaking documentation/typing clarification.
- **Error handling:** Added a normalized Web App error adapter that supports
  both legacy string errors and the documented structured envelope.
