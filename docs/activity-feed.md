# Activity Feed

Trip activity is the private audit-style timeline for collaboration and planning changes.

## Endpoints

- `GET /trips/{tripId}/activity`
- `GET /trips/{tripId}/activity/stream`

The stream emits server-sent events for clients that keep the trip detail page open. The list endpoint is the durable fallback.

## Collaboration Events

Trip collaboration v1 records events for:

- collaborator invited, accepted, declined, removed, revoked
- comment created, updated, deleted, resolved, reopened
- suggestion created, accepted, rejected, resolved
- vote added or removed
- ownership transferred

Activity metadata avoids comment body text and other high-sensitivity payloads. It includes stable IDs, target types, target IDs, roles, and status transitions needed for UI refresh and audit display.
