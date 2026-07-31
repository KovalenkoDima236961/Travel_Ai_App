# Trip Workspace navigation architecture

## Canonical routes

Primary sections use `/trips/{tripId}/{section}`. Optional `view` and entity parameters keep context shareable:

```text
/trips/{tripId}/plan?view=timeline&day=3&item={itemId}
/trips/{tripId}/money?view=expenses&expense={expenseId}
/trips/{tripId}/group?view=discussion&comment={commentId}
/trips/{tripId}/prepare?view=checklist&item={checklistItemId}
/trips/{tripId}/more?view=versions&version={versionId}
```

The route model and compatibility map are centralized in `apps/web/src/lib/trip-workspace/navigation.ts`. Entity-to-DOM resolution is in `deep-link.ts`. UI components must not parse these parameters independently.

## Compatibility table

| Historical input | Canonical interpretation |
| --- | --- |
| `/trips/{id}` | Overview |
| `?tab=command_center` | Overview summary |
| `?tab=itinerary|timeline|calendar` | Plan corresponding view |
| `?tab=route&legId={id}` | Plan route, `leg={id}` |
| `?tab=budget|expenses|receipts|settlements` | Money corresponding view |
| `?tab=team|collaborators|comments|polls|dates|activity` | Group corresponding view |
| `?tab=checklist|reminders|offline` | Prepare corresponding view |
| `?tab=health|policy|versions` | More corresponding view |

Historical URLs are interpreted in place to protect email and browser-bookmark history. New navigation, Global Search selections, and notification navigation emit canonical routes. Query normalization never accepts absolute external redirects, so it cannot become an open redirect.

## Resolution and focus

Resolution has the states `resolving`, `resolved`, `not_found`, `forbidden`, `feature_disabled`, and `offline_unavailable`. After section/view resolution, the page loads the target, scrolls it into view, applies a visible focus ring for 2.4 seconds, and focuses a temporary `tabindex=-1` target. Pointer or keyboard interaction clears the highlight; reduced-motion users receive instant scrolling. A missing entity remains inside the valid trip shell and offers section/overview navigation without disclosing inaccessible metadata.

Canonical and legacy entity parameters are both accepted during migration. Stable target IDs are required for itinerary items, route legs/stops, expenses, receipts, checklist items, reminders, decisions, collaborators, activity, comments, health issues, and versions.

## Desktop and mobile

Desktop shows the six primary sections in one labeled navigation landmark and the selected section uses `aria-current=page`. Subviews are a separate, wrapping secondary landmark.

Mobile shows the current section in a sticky section picker, keeps Global Search adjacent, and uses five persistent bottom destinations: Overview, Plan, Money, Prepare, and More. Group remains one tap away in the picker; when selected, More is the persistent escape destination. Safe-area padding and 44-pixel minimum targets are applied.

## Context and unsaved data

Shareable context belongs in the URL. Schedule view/day are also passed to the existing scheduling workspace; ephemeral filters and map viewport may remain local. Existing edit state is not remounted for same-section view changes. Primary/subview links warn before an active unsaved itinerary edit is discarded, and `beforeunload` protects browser close/reload.

## Public and authorization boundaries

`/share/{shareToken}` is not a workspace route. Every private section loads the trip before entity resolution and uses backend-returned capabilities for UI selection. Backend authorization remains mandatory for every mutation and entity request.
