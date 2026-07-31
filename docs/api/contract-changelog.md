# API contract changelog

## Unreleased

- **Trip service global search:** Documented `GET /search`, result type filters,
  archive inclusion, safe command opt-in, `data`/`pagination` response aliases,
  matched fields, query metadata, and search-specific validation/rate-limit
  errors. Additive for existing `items`/`groups` clients.
- **Trip service alpha:** Added closed-alpha waitlist, invite activation,
  participant metadata, product analytics event, and structured feedback
  contracts. Additive.
- **Trip service:** Added safe public feature-flag projection, ops flag
  management/audit routes, and the `feature_disabled` error code. Additive.
- **AI model serving:** Added Trip Service user feedback submission for online
  model comparisons, ops deployment registration/pause/rollback/summary
  routes, and AI Planning optional `deploymentKey`, `requestAssignmentId`, and
  `inferenceMode` routing metadata on itinerary generation. Additive and
  backend-owned.
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
