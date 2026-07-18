# Mobile & responsive audit

Last updated: 2026-07-18

This is a frontend quality pass over existing flows. It preserves routes, API behavior, and the current Tailwind component system.

## Screens audited

- Global layout, shared controls, loading/error/empty states, notifications dropdown, and mobile dialog behavior
- Create Trip (`/trips/new`)
- Trips (`/trips`) and Trip Detail (`/trips/[id]`), including its section navigation, itinerary, route, checklist, reminders, budget, and expenses areas
- Notifications (`/notifications`) and notification preferences in Settings
- Templates, Workspaces, and Settings
- Cost analytics and workspace budget detail views

## Issues found and fixes made

| Area | Issue | Fix |
| --- | --- | --- |
| Global layout | Long content and embedded media could escape narrow viewports; compact inputs can trigger iOS zoom. | Added safe horizontal clipping at root level, `min-width: 0` for the main region, responsive media sizing, 16px mobile form text, visible focus styles, and reduced-motion support. |
| App chrome | Header branding and 36–38px account/notification controls were cramped on phones. | Reduced mobile gutters, hid decorative wordmarks below `sm`, and increased account, notification, and shared action controls to 44px. |
| Trip Detail | The full left navigation rail appeared above content on tablet/phone and made the first viewport dense. | Replaced it below `xl` with a horizontally scrollable 44px section switcher; left-rail summary cards remain desktop-only. |
| Forms and dialogs | Add Expense was an inline, long form with actions easily pushed below the keyboard. | Added `FullScreenMobileDialog`; Add Expense is now a focus-trapped full-screen mobile workflow with a persistent safe-area-aware action bar and constrained desktop dialog. |
| Dense data | Per-traveler allocations and the main cost analytics/workspace budget tables were desktop tables on every viewport. | Added `ResponsiveDataView` and mobile summary cards while retaining desktop tables from `md` upward. |
| Specialized tables | Route comparison, transport comparison, and notification delivery settings need more columns than a phone supports. | Preserved their horizontal tables but made the regions keyboard-focusable and added a visible mobile “scroll sideways” hint. |
| Checklist and reminders | Checklist completion used a small unlabeled checkbox. | Wrapped it in a labelled 44px touch target; existing reminder cards/actions use the updated shared control sizing. |
| Maps | The full itinerary map claimed too much vertical space on small screens. | Reduced its default mobile height to 260px while retaining the desktop 420px map and its list-based itinerary context. |
| Notifications | The header dropdown could be awkwardly positioned or too tall on narrow screens. | It becomes a viewport-safe sheet-like overlay on phones with an independently scrollable list. |

## Remaining limitations

- Several older, feature-specific dialogs still use local shells. They should be migrated to `FullScreenMobileDialog` incrementally when those flows are next changed, rather than in a broad behavioral rewrite.
- Operations-only tables remain scrollable desktop data views; they are not primary phone workflows.
- This pass has source-level and automated validation. Final device QA should cover iOS Safari keyboard behavior, Android Chrome, PWA install/update banners, Leaflet map gestures, and all four translated locales with long content.
- The current test suite has two unrelated offline-cache mock failures (`offline-storage.test.ts` expects stores that the cache implementation now accesses). They are recorded separately from this UI pass.
