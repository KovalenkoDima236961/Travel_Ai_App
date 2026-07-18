# API endpoint inventory

This is a practical inventory of browser-facing and operational route groups.
Paths are exact unless a `{placeholder}` appears. For DTO field detail, inspect
the handler DTO/source listed in [API overview](overview.md) before implementing
a client. Private routes require bearer JWT unless stated otherwise.

## Auth and profile

| Method/path | Service | Auth / permission | Request and response | Important errors |
| --- | --- | --- | --- | --- |
| `POST /auth/register`, `/auth/login` | Auth | Public | Email/password → user and token pair | validation, unauthorized, rate limit |
| `POST /auth/refresh`, `/auth/logout` | Auth | Refresh token body | Rotate/revoke refresh token | unauthorized, rate limit |
| `GET /auth/me` | Auth | Bearer | Current identity | unauthorized |
| `GET/PUT /users/me/profile` | User | Bearer | Current profile | validation, unauthorized |
| `GET/PATCH /users/me/preferences` and `/completeness` | User | Bearer | Preferences/completeness | validation |
| `POST/GET /users/me/export...`, account cleanup request | User | Bearer | Private export/cleanup state | forbidden, not found |
| `/workspaces/*`, `/workspace-invitations/*` | User | Bearer; workspace role | Workspace, membership, invitation CRUD | forbidden, conflict |

## Trips and itinerary

| Method/path | Service | Permission | Request and response | Important errors |
| --- | --- | --- | --- | --- |
| `POST/GET /trips`, `GET /trips/{id}`, `/shared-with-me`, `/library` | Trip | Private/read access | Create/list/read scoped trip DTOs | validation, forbidden |
| `POST /trips/{id}/archive`, `/restore`; export routes | Trip | Owner/workspace role | Archive/restore and private export job | forbidden, conflict |
| `PUT /trips/{id}/itinerary` | Trip | Owner/editor | Itinerary + `expectedItineraryRevision` → updated trip | itinerary conflict |
| Version/reaction/travel-status/day/item regeneration routes under `/trips/{id}/itinerary/*` | Trip | Owner/editor or allowed viewer read | Version history, reactions, revision-safe edits/jobs | forbidden, conflict |
| `GET/PUT /trips/{id}/route`; route-leg transport search/selection | Trip | Read/edit by action | Route and estimated transport options | validation, provider unavailable, conflict |
| `GET/PUT/DELETE /trips/{id}/accommodation` | Trip | Read/edit by action | Accommodation state | validation, conflict |
| `GET /trips/{id}/command-center-summary`, `/health`, `/verification`, `/travel-day` | Trip | Private read | Compact/advisory summaries | forbidden |
| `POST /trips/{id}/verification/actions` | Trip | Owner/editor | Explicit verification refresh | provider quota/unavailable |

## Generation, discovery, templates, and Copilot

| Method/path | Service | Permission | Request and response | Important errors |
| --- | --- | --- | --- | --- |
| `POST /trips/{id}/generate`, `/generation-jobs`; `GET .../generation-jobs/{jobId}`; cancel | Trip | Owner/editor | Create/read/cancel async generation | conflict, generation failed |
| Budget optimization and repair job/proposal routes | Trip | Owner/editor | Queued proposal then revision-safe apply/discard | conflict, generation failed |
| `POST /trip-discovery/suggestions`, `/surprise-me`, sessions/refine/vote | Trip | Private | Discovery sessions and votes | validation, provider/AI failure |
| `POST /route-alternatives/suggest` and session/trip alternative routes | Trip | Private/read-edit by action | Route alternatives/create/apply/poll | conflict |
| `GET/POST/PATCH /trip-templates/*`; template adaptation jobs | Trip | Private/workspace role | Template CRUD/adaptation/create trip | forbidden, conflict |
| `POST /trips/{tripId}/copilot/chat` | Trip | Private trip access | Safe Copilot response | validation, generation failed |

## Collaboration, planning, and business domains

| Method/path | Service | Permission | Request and response | Important errors |
| --- | --- | --- | --- | --- |
| Collaborators/invitations/travelers under `/trips/{id}` | Trip | Owner/editor or accepted participant | Invite, role, acceptance, traveler CRUD | forbidden, conflict |
| Comments/activity/presence/edit-lock routes | Trip | Trip access; writes role-gated | Comment CRUD, SSE, advisory presence/locks | edit lock conflict |
| Polls, date options, availability, group preferences/readiness | Trip | Read/edit by action | Decisions, calendar import and nudges | forbidden, conflict |
| Checklist/reminder routes and `/reminders/assigned-to-me` | Trip | Read/edit/assignee by action | Generate and manage private preparation tasks | validation, forbidden |
| Budget, analytics, cost splitting, expenses, receipts, settlements | Trip | Read/edit by action | Financial state, CSVs, multipart receipt upload | upload errors, conflict |
| `GET/POST/PATCH/DELETE /trips/{id}/share`; `/public/trips/{shareToken}*` | Trip | Owner for private; public for share | Share controls/status/unlock/sanitized view | expired/password/rate limit |
| Workspace policy, approval, budget, analytics routes | Trip | Workspace owner/admin/member by action | Governance and shared financial views | forbidden, conflict |

## Notifications, providers, AI, and operations

| Method/path | Service | Auth / permission | Request and response | Important errors |
| --- | --- | --- | --- | --- |
| `/notifications`, unread/read, preferences, mutes, digests, push, stream | Notification | Bearer | In-app lifecycle, SSE and delivery settings | unauthorized, validation |
| `POST /internal/notifications/batch`, `/process-digests` | Notification | Internal token | Trusted fan-out/digest work | internal auth required |
| `/places/*`, `/routes/estimate`, `/weather/forecast`, `/exchange-rates/*`, prices/availability/transport | External | Bearer where route middleware applies | Normalized/mock provider estimates | provider errors |
| Calendar connect/status/free-busy/disconnect and callback | External | Bearer except OAuth callback | OAuth and calendar data controls | unauthorized, provider unavailable |
| `/ops/providers/*` | External | Ops/internal configuration | Provider status/quota/reset-dev | forbidden, quota |
| `/generate-itinerary`, checklist, regenerate, optimize, repair, adapt, discovery, Copilot, recap, knowledge/destination context | AI | Service-to-service deployment boundary | Strict validated AI schemas | validation, generation failed |
| `/ops/jobs*`, `/ops/ai-generations*` | Trip | Ops authorization | Job/trace list, retry/cancel/mark failed | forbidden, conflict |
| `GET /feature-flags/public`; `/ops/feature-flags*` | Trip | Public safe projection; ops routes require allowlisted admin | Browser-safe booleans; list/change/reset/audit reviewed runtime controls | feature disabled, forbidden |
| `/ops/worker/status`, queues and DLQ routes | Worker | Ops authorization | Worker/queue/DLQ diagnostics and action | forbidden, conflict |
| `/health`, `/ready`, `/version`, `/metrics` | Go services and AI; Web exposes version at `/api/version` | Public local/monitoring | Liveness, dependencies, non-sensitive build metadata, metrics | service unavailable |

## Internal endpoints

Known internal groups are Auth `/internal/users/*`, User `/internal/workspaces/*`,
Trip `/internal/reminders/process-due` and `/internal/data-exports/account-package`,
Notification `/internal/notifications/*`, and External calendar sync/delete.
They require internal authentication and should only be called by named service
clients. Their request shapes are intentionally not public frontend contracts.

## Related docs

- [API overview](overview.md)
- [Service README files](../../services)
- [Endpoint implementation playbook](../development/playbooks/add-backend-endpoint.md)
