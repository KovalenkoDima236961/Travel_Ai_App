# Frontend performance baseline

Date: 2026-07-18. This is a code-path baseline, not a device-lab timing claim. The production analyzer ran successfully on this revision; keep recording output before setting route-specific byte limits.

## Production build observations

`npm run analyze` completed with a 103 kB shared first-load baseline. Largest measured routes were private trip detail (123 kB route / 402 kB first load), create trip (31.8 kB / 312 kB), settings (21.5 kB / 289 kB), and analytics routes (about 240–247 kB first load). The analyzer reports are generated under `.next/analyze/` and are intentionally build artifacts, not committed documentation.

## High-risk routes and observations

| Route | Initial-risk observation | Current strategy |
| --- | --- | --- |
| `/` and `/trips` | Shared providers previously mounted the full command-palette controller. | A small shortcut listener now imports the controller only after Cmd/Ctrl+K. |
| `/trips/new` | Planning, route, and preference UI are client-side; it must not pull map or PDF renderers. | Leaflet remains behind map components and PDF is interaction-loaded. |
| `/trips/[id]` | The largest private route: itinerary tools, summaries, dialogs, map, collaboration, and Copilot. | Core trip plus Command Center load first; sections use observer/deep-link activation and heavy panels/dialogs are dynamic. |
| `/library` | Historical trip data can be long-lived. | API uses compact records and filters; retain page-size limits. |
| `/notifications` | Unread state is global and easy to over-poll. | SSE is primary; count polling is slow fallback and the list is cursor-paged. |
| `/ops` | Several operational queries refresh together. | Polling pauses in hidden documents and keeps a single bounded cadence. |

## Known heavy boundaries

- Leaflet and `react-leaflet` load only when a map component is mounted. Leaflet CSS is no longer part of the root layout.
- Copilot panel, map rail, route/tools panels, optional dialogs, and itinerary history use dynamic boundaries with compact fallbacks.
- Browser PDF generation is imported only after the user presses **Download PDF**.
- The command-palette registry, current-trip lookup, local result transformation, and dialog renderer load only when the palette opens.

## Query and polling baseline

- Global query defaults are 30 seconds with focus refetch disabled.
- Preferences are five minutes; weather is ten minutes; health, verification, and confidence are 45 seconds; list/detail hooks declare narrower settings where appropriate.
- Terminal jobs never poll. Active generation, repair, template-adaptation, and export jobs pause while `document.hidden`; after the first four updates their cadence backs off from 2.5/1.5 seconds to five seconds.
- Notifications use the unread-count endpoint, not the full list, and SSE invalidates cached notification data.

## Offline/PWA baseline

- `cachedTrips` is a compact, user-indexed summary store. Full itinerary/route records live in `cachedTripDetails` and are read only when a trip opens.
- Snapshot detail and summary writes are committed in one IndexedDB transaction, and unchanged revisions are skipped.
- The service worker only caches app-shell and `/_next/static/` assets. Runtime static assets are capped at 120 entries; private APIs and receipt files are not cached by the worker.

## Working budgets

These are guardrails for profiling and review, rather than brittle automated timing assertions.

| Area | Budget / expectation |
| --- | --- |
| Home | Keep initial JavaScript limited to shell/auth/i18n essentials; no map, PDF, Copilot, or command-palette controller. |
| Trips list | First render under two seconds with warm local services; render the first bounded response only. |
| Trip detail | Show title, status, and compact Command Center before optional panels. Do not request/render inactive heavy sections. |
| Create trip | No map/chart/PDF bundle before the relevant interaction. |
| Library | First compact page only; do not hydrate historical details, receipts, or comments. |
| Notifications | Unread badge must not fetch the full notification list. |
| Offline | Offline list reads compact summaries; open-trip details are on demand. |

Run `cd apps/web && npm run analyze` to produce the Next bundle report, then attach route sizes to the relevant release or performance review.
