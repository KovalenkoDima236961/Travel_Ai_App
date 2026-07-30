# Trip Collaboration

Trip collaboration v1 is owned by Trip Service and is private-only. Public share routes never expose collaborators, invitations, comments, activity, suggestions, votes, or member presence.

## Capabilities

| Capability | API surface | Notes |
| --- | --- | --- |
| Email invitations | `/trips/{tripId}/invitations` | Owners create, list, resend, revoke, accept, and decline invitations. Registered users are linked immediately; email-only invites can be accepted later by a matching signed-in user. |
| Members | `/trips/{tripId}/members` | Returns the owner plus active or pending collaborators with a computed permission map. Non-managers do not receive member email addresses. |
| Ownership transfer | `/trips/{tripId}/members/transfer-ownership` | Only the current owner can transfer a personal trip to an accepted collaborator. Workspace-owned trips cannot transfer direct ownership. |
| Comments | `/trips/{tripId}/comments` | Comments can target trips, days, itinerary items, budgets, routes, and attachments. Item comments remain queryable by day/item for existing UI badges. |
| Suggestions | `/trips/{tripId}/suggestions` | Collaborators can propose trip changes. Editors/owners can accept, reject, or resolve suggestions. Accepted itinerary-item suggestions use the normal itinerary revision guard. |
| Votes | `/trips/{tripId}/votes` | Collaborators can vote on activities, restaurants, hotels, destinations, and suggestions. Responses are aggregated by target and include the current user's vote. |
| Activity | `/trips/{tripId}/activity` and `/activity/stream` | Collaboration mutations emit audit-style activity events and optional notifications. |

## Storage

Migration `000048_add_trip_collaboration_v1` adds `trip_invitations`, `trip_suggestions`, `trip_votes`, and `trip_presence`, and extends `trip_collaborators` and `itinerary_comments`.

## Observability

`trip_collaboration_events_total{action,result}` records low-cardinality collaboration outcomes such as invite creation, invite acceptance, comment creation, suggestion status changes, votes, member leave, and ownership transfer.

## Frontend

The trip detail tools section includes:

- `CollaboratorsPanel` for email invites, pending invite management, members, and ownership transfer.
- `ItemCommentsPanel` for item threads, replies, and resolve/reopen.
- `TripPlanningCollaborationPanel` for trip-level suggestions and quick votes.
