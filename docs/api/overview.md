# API overview

This repository currently has hand-maintained HTTP contracts. Handler request
and response DTOs are the executable source of truth; the endpoint inventory
below is the navigation layer. There is no repository-wide generated OpenAPI
artifact in v1, so do not claim an OpenAPI client is authoritative yet.

## Conventions in use

- JSON uses camelCase field names in browser-facing DTOs. UUIDs are canonical
  string IDs; timestamps are RFC 3339 strings; date-only values are ISO-8601
  `YYYY-MM-DD`.
- Authenticated endpoints expect `Authorization: Bearer <access JWT>`. Auth
  refresh/logout use their documented refresh-token body. User identity comes
  from the JWT, never a caller-supplied `userId`.
- `/internal/*` routes require `X-Internal-Service-Token` and are for trusted
  services only. A caller should also supply the existing internal service-name
  header where supported. Do not call them from the browser.
- Request/correlation middleware accepts or creates `X-Request-ID` and
  `X-Correlation-ID`; logs and responses propagate identifiers where the
  service middleware supports them.
- Existing success bodies are domain-specific and frequently raw objects or
  arrays. Pagination is endpoint-specific: cursor lists return a cursor/has-more
  style continuation, while expense/receipt lists use `limit`, `offset`, and
  `nextOffset`. Read the handler DTO before changing a client.
- Mutation idempotency is not globally standardized. Generation jobs and other
  asynchronous workflows use persisted state/correlation behavior rather than
  a universal `Idempotency-Key`. Do not invent client mutation IDs without a
  service contract.
- Itinerary-changing requests use `expectedItineraryRevision`. A stale write is
  a conflict: refetch, present/merge the latest state, then submit explicitly.
- Public share routes return a deliberately sanitized, read-only DTO. Do not
  reuse private trip DTOs for public views.

## Error behavior

Go handlers use service-specific error mappers and some legacy paths return a
simple message rather than a universal error envelope. Clients must preserve
HTTP status and request ID and map known errors conservatively. See
[errors](errors.md); it records a target vocabulary, not a claim that every
legacy endpoint already emits identical JSON.

## Rate limits and security

Login/register/refresh, public share unlock/view, receipt uploads, provider
requests and queues have bounded rate controls. Expect `429` where enabled;
respect `Retry-After` if present and show a retryable UI state. Never place
JWTs, refresh tokens, internal tokens, calendar tokens, provider keys, or raw
prompt/receipt data in URLs or logs.

## Endpoint source locations

- Auth: `services/auth-service/internal/httpserver/handler/auth.go`
- User/workspace: `services/user-service/internal/httpserver/handler/` and `internal/workspaces/handler.go`
- Trip/public/internal: `services/trip-service/internal/http-server/handler/trip.go`
- Notifications: `services/notification-service/internal/httpserver/handler/`
- Providers/calendar: `services/external-integrations-service/internal/httpserver/routes.go`
- AI: `services/ai-planning-service/app/api/routes.py`

## Related docs

- [Endpoint inventory](endpoint-inventory.md)
- [Errors](errors.md)
- [Service boundaries](../architecture/service-boundaries.md)
