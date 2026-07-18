# Frontend performance guide

## Route and bundle boundaries

Keep App Router layouts and page shells small. Client-only interaction belongs in the smallest possible client component. Use `next/dynamic` for map renderers, chart/analytics panels, PDF/export code, receipt preview/OCR review, Copilot, command-palette controllers, and dialogs that are closed on first paint. Every dynamic component needs a fixed-height fallback where it occupies visible layout.

Do not dynamically import navigation, the trip title/status, a primary form, authentication-critical controls, or a page skeleton merely to reduce a number in the analyzer.

Run the optional analyzer with:

```bash
cd apps/web
npm run analyze
```

`ANALYZE=true` is intentionally isolated from normal builds.

## TanStack Query rules

- Use the canonical factories in `src/lib/query-keys.ts`; keys include IDs and normalized filter primitives.
- Give data an intentional freshness window. Start from the provider default (30 seconds), use minutes for preferences/static library data, and declare shorter freshness only for actively changing views.
- Set `enabled` until required IDs, permissions, and visible/deep-linked sections are available.
- Mutations invalidate their affected trip subtree and explicit summary dependents; never invalidate every trip, notification, or settings query as a convenience.
- Poll only active jobs. Stop at terminal status, pause when the document is hidden, and back off long-running work. Prefer notification SSE to list polling.

## Lists and render work

Request an explicit first page/limit. Prefer backend pagination or load-more controls for notifications, activity, comments, library, expenses, receipts, and operations tables. Keep mobile card conversions bounded too.

Memoize only measured or obvious hotspots: sorting/grouping large data, itinerary normalization, derived cost totals, and repeated item rows. Avoid repeatedly allocating row callbacks/objects when a stable ID plus memoized handler will do. Do not use `useMemo` around trivial formatting.

## Offline and PWA rules

- Keep compact trip summaries separate from full offline details. Read detail only when the user opens that trip.
- Batch related IndexedDB writes in one transaction and skip unchanged revision/update records.
- Preserve user scoping and clear local data on logout using the existing helpers.
- Service workers cache only bounded shell/static assets. Do not cache authenticated API responses, receipt files, or user content in the worker.

## Review checklist for a new screen

1. Does its route avoid importing maps, charts, PDF/export, rich dialogs, or Copilot before interaction?
2. Are queries keyed, gated, and given an intentional stale time?
3. Is polling terminal-aware and visibility-aware?
4. Is a long list paged/windowed and are row transforms bounded?
5. Are sensitive route values, search text, and trip data excluded from metrics/logs?
6. Does the offline path use summaries first and avoid repeated full-cache writes?
7. Does `npm run typecheck`, `npm run test`, `npm run build`, and the relevant manual smoke flow pass?
