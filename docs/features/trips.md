# Trips

Trip Service owns the private planning aggregate. A trip can be personal or
workspace-scoped, active or archived, and is always accessed through its owner,
accepted collaborator role, or workspace role.

## Lifecycle and revisions

- Create/list/read through `/trips`; active planning is separate from the
  archive/library view.
- An itinerary mutation includes `expectedItineraryRevision`. The service
  rejects stale edits rather than overwriting a collaborator's newer version.
  Refetch, show/merge conflict, and submit again deliberately.
- Versions can be listed, viewed, and restored through itinerary version routes.
  Presence and edit locks are advisory collaboration signals, not permission.

## Major domains

| Domain | Notes |
| --- | --- |
| Route/accommodation | Ordered stops, legs, selected transport estimates, and accommodation anchors; no bookings are created. |
| Budget/expenses | Budget, estimated/actual costs, cost splitting, settlements, confidence, analytics and CSV. Money is an estimate unless supplied as an actual expense. |
| Collaboration | Owner/editor/viewer plus workspace permissions; comments/activity/polls and invitations are server-authorized. |
| Checklist/reminders | Private preparation records, generation/manual edits, assignment, worker processing, and notification integration. |
| Public share | Read-only sanitized token endpoint, optional expiry/password, explicit owner controls. |
| Library/templates/recap | Archive is non-destructive; private/workspace templates and post-trip recap use separate permissions. |
| Governance | Workspace policies/approval and advisory health, verification, group-readiness, and command-center data. |

## Safe extension rules

Keep Trip Service authoritative for trip access and revision writes. Use User
Service internal workspace lookup instead of copying workspace membership;
use External Integrations/AI/Notification clients instead of direct data access.
For a new route, follow the [backend endpoint playbook](../development/playbooks/add-backend-endpoint.md).

## Related docs

- [Key flows](../architecture/key-flows.md)
- [Endpoint inventory](../api/endpoint-inventory.md)
- [Receipts and expenses](receipts-expenses.md)
