# Key flows

These diagrams show responsibility boundaries rather than every request field.

## Register, login, and refresh

```mermaid
sequenceDiagram
  participant W as Web App
  participant A as Auth Service
  participant DB as Auth DB
  W->>A: POST /auth/register or /auth/login
  A->>DB: create/find user; store/validate refresh token
  A-->>W: access JWT + rotating refresh token
  W->>A: POST /auth/refresh
  A->>DB: revoke old refresh token; store replacement
  A-->>W: new access + refresh token
```

## Create a trip

```mermaid
sequenceDiagram
  participant W as Web App
  participant T as Trip Service
  participant U as User Service
  participant DB as Trip DB
  W->>T: POST /trips (Bearer JWT)
  T->>U: internal workspace access check when workspace-scoped
  U-->>T: role decision
  T->>DB: create trip at initial itinerary revision
  T-->>W: private trip DTO
```

## AI generation job lifecycle

```mermaid
sequenceDiagram
  participant W as Web App
  participant T as Trip Service
  participant Q as RabbitMQ
  participant R as Worker
  participant AI as AI Planning
  participant N as Notification
  W->>T: POST /trips/{id}/generation-jobs (expected revision)
  T->>Q: publish job after persistence
  T-->>W: queued job
  Q->>R: deliver job
  R->>T: fetch/update guarded job state
  R->>AI: generate/repair strict JSON
  AI-->>R: validated result or controlled error
  R->>T: save only if revision/permissions still valid
  R->>N: internal batch notification when appropriate
  T-->>W: job status via poll/query refresh
```

## Itinerary conflict, public share, and collaboration

```mermaid
sequenceDiagram
  participant E as Editor
  participant T as Trip Service
  participant DB as Trip DB
  E->>T: PUT itinerary + expectedItineraryRevision
  T->>DB: compare-and-write revision
  alt current revision
    DB-->>T: incremented revision
    T-->>E: updated itinerary
  else stale revision
    DB-->>T: conflict
    T-->>E: conflict; refetch/merge before retry
  end
```

```mermaid
sequenceDiagram
  participant Owner
  participant T as Trip Service
  participant Viewer
  Owner->>T: POST /trips/{id}/share (expiry/password optional)
  T-->>Owner: share token
  Viewer->>T: GET /public/trips/{token}/status
  opt password protected
    Viewer->>T: POST /public/trips/{token}/unlock
  end
  Viewer->>T: GET /public/trips/{token}
  T-->>Viewer: sanitized read-only DTO
```

Collaborator invitations are created by a permitted trip owner/editor, linked
to an existing identity lookup, and accepted/declined through authenticated
trip routes. Permissions are checked on every subsequent request; a UI role
badge never grants access.

## Related docs

- [Trips feature guide](../features/trips.md)
- [AI generation guide](../features/ai-generation.md)
- [API overview](../api/overview.md)
