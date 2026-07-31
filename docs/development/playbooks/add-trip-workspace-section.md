# Add a Trip Workspace section or view

1. Prefer a view under an existing primary section. A seventh primary section requires product/UX review and a mobile-navigation decision.
2. Add the view to `TRIP_WORKSPACE_SECTIONS` and the visible subview list; add all four locale keys.
3. Map any historical tab in `LEGACY_TRIP_TAB_MAP`, add required query-load anchors, and keep entity parameters stable.
4. Reuse an existing domain panel/query. Do not copy business rules or mount every section eagerly.
5. Apply capability, archive, offline, and feature-flag gates before rendering mutation controls; backend enforcement is still required.
6. Add route, compatibility, viewer, mobile, keyboard, empty/error/offline, and deep-link tests. Update the mapping docs.
