# Trip information architecture

Trip Workspace answers four questions in order: what is happening, what should I do next, where is a capability, and what requires attention.

| Area | User intent | Primary contents |
| --- | --- | --- |
| Overview | Orient and act | identity/status, next action, attention, readiness, upcoming/recent context, quick actions |
| Plan | Build and use the itinerary | itinerary, schedule modes, route/transport, stay, map, place verification |
| Money | Understand and settle costs | budget, expenses, receipts, splits, settlements |
| Group | Coordinate people and decisions | people/access, discussion, availability, decisions, approvals, activity |
| Prepare | Become departure-ready | checklist, reminders, offline readiness, existing documents capability |
| More | Use infrequent/admin tools | share/export, history, recap, template reuse, health/policy detail, archive/restore |

Overview prioritizes action over metrics and omits empty cards. More is grouped, permission-aware, and cannot be the sole home of a critical warning. Mobile persistent navigation contains no more than five destinations; Group is available through the current-section picker.

Lifecycle changes emphasis without creating another shell. Active trips retain today/route/offline/expense access; completed trips emphasize recap and settlement; archived trips are explicitly read-only with export/history/template/restore. Public shares remain a separate sanitized surface.
