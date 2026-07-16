# Access Control Matrix

Legend: **R** read, **W** mutate, **Own** only the actor's contribution,
**M** manage, **—** denied. Public means the dedicated sanitized public DTO only.
Workspace roles apply only to a workspace trip that the existing membership
access check grants; membership never grants access to personal trips.

| Resource | Owner | Collab editor | Collab viewer | WS owner | WS admin | WS member | WS viewer | Public share | Internal service | Ops admin | Anonymous | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Trip | R/W/delete | R/W | R | R/W/delete | R/W/delete | R/W | R | R sanitized | — | — | — | Pending/removed/random users denied. |
| Itinerary | R/W | R/W | R | R/W | R/W | R/W | R | R sanitized | — | — | — | Public has no debug/version metadata. |
| Route/transport | R/W | R/W | R | R/W | R/W | R/W | R | R sanitized | — | — | — | Provider search remains private. |
| Budget | R/W | R/W | R | R/W | R/W | R/W | R | — | — | — | — | Financial private. |
| Budget confidence | R | R | R | R | R | R | R | — | — | — | — | Never in public output. |
| Expenses | R/W | R/W | R, Own W | R/W | R/W | R/W | R, Own W | — | — | — | — | Existing creator checks constrain viewer mutation. |
| Receipts/OCR | R/W | R/W | R, Own upload/delete | R/W | R/W | R/W | R, Own upload/delete | — | — | — | — | File endpoint always reauthorizes. |
| Settlements | R/W | R/W | R/Own action | R/W | R/W | R/W | R/Own action | — | — | — | — | Financial private. |
| Comments | R/W | R/create | R/create/Own W | R/W | R/W | R/create | R/create/Own W | — | — | — | — | React escapes text; no raw HTML rendering. |
| Activity | R | R | R | R | R | R | R | — | — | — | — | Safe metadata only. |
| Group readiness | R/W nudge | R/W nudge | R/Own input | R/W | R/W | R/W | R/Own input | — | — | — | — | Not public. |
| Trip health | R | R | R | R | R | R | R | — | — | — | — | Debug fields remain private. |
| Command center | R | R | R | R | R | R | R | — | — | — | — | Composite private view. |
| Approval | R/act | R/submit | R/submit | R/act | R/act | R/submit | R | — | — | — | — | Workspace policy further constrains action. |
| Policy | R/W | R | R | R/W | R/W | R | R | — | — | — | — | Workspace policy is authoritative. |
| Workspace budgets | R/W | — | — | R/W | R/W | R subject to policy | R | — | — | — | — | Workspace-scoped resource. |
| Public share management | M | — | — | M | M | — | — | — | — | — | — | Tokens/password hashes never returned in public DTO. |
| Public share view | R | R | R | R | R | R | R | R sanitized | — | — | R sanitized | Must be enabled, unexpired, and unlocked. |
| Exports | R | R | R | R | R | R | R | public-specific only | — | — | — | Export builders must use the same private/public DTO. |
| Notifications | own | own | own | own | own | own | own | — | create batch | — | — | User ID comes from JWT. |
| Calendar sync/free-busy | own R/W | own R/W | own R/W | own R/W | own R/W | own R/W | own R/W | — | provider internal | — | — | Only sanitized availability summary leaves integration service. |
| Internal endpoints | — | — | — | — | — | — | — | — | R/W scoped route | — | — | Token required; no browser JWT. |
| Ops endpoints | — | — | — | — | — | — | — | — | optional scoped call | R/act | — | JWT email allowlist; no owner shortcut. |

The central policy enumerates trip, itinerary, route, budget, expense, receipt,
comment, activity, share, collaborator, approval, policy, health, command-center,
readiness, and ops permissions. High-risk receipt handlers use these decisions
directly. Existing domain services retain creator/assignee checks for “Own”
operations and workspace services retain role/policy checks.

