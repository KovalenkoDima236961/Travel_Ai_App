# Workspaces, approvals, and policies

User Service owns workspace identity and membership. The roles are `owner`,
`admin`, `member`, and `viewer`; invitations have an explicit pending/accepted/
declined lifecycle. Trip Service asks User Service's protected internal lookup
routes when it must authorize workspace-scoped trip behavior.

## Responsibilities

| Area | Owner | Rule |
| --- | --- | --- |
| Workspace/membership/invitation | User Service | Identity and role source of truth; no client-supplied role trust. |
| Workspace trips, budgets/templates, policies | Trip Service | Enforces returned workspace role on each request. |
| Approval/risk/repair | Trip Service | Policy evaluation/approval is deterministic and revision-aware; AI advice is not authority. |
| Notifications | Notification Service | Invites/approval events respect recipients and preferences. |

Policies can evaluate planned trips and surface warnings/blocking conditions;
approval routes submit, approve, request changes, cancel, and list events.
Repair proposals are reviewed and applied only with correct permissions and an
expected current itinerary revision. Risk scores and AI suggestions are
advisory—never a bypass for policy or permission checks.

## Extension rules

Add workspace permissions in backend tests for owner/admin/member/viewer and
cross-workspace isolation. Use service-internal access lookup instead of copying
membership tables. Update the threat model when adding policy rules involving
financial, privacy, sharing, or external data effects.

## Related docs

- [Service boundaries](../architecture/service-boundaries.md)
- [Trips](trips.md)
- [Security-sensitive feature playbook](../development/playbooks/add-security-sensitive-feature.md)
