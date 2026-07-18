# Playbook: add a frontend page

1. Add the App Router route under `apps/web/src/app`; use a server component by default and add a client boundary only for browser state/interactions.
2. Reuse `src/lib/api`, hooks, query keys, and UI primitives. Do not fetch a private service directly from a component when an existing same-origin proxy boundary applies.
3. Provide loading, error, empty, and permission-denied states. Use the shared loading/error/empty primitives described in `docs/frontend/ux-guidelines.md`.
4. Add every user-visible string to `messages/en.json` and translate the `es`, `uk`, and `fr` catalogs. Use locale formatters for dates/money.
5. Meet responsive/mobile and keyboard/accessibility requirements: semantic headings, labelled controls, focus management, clear error association, and no color-only status.
6. Add focused Vitest/Testing Library coverage for state and permissions. Add a Playwright scenario when the route changes a high-value cross-service flow.
7. Run `npm run lint`, `npm run typecheck`, `npm test`, and `npm run build` (or `./scripts/test-frontend.sh`).

Related: [frontend README](../../../apps/web/README.md), [query hook playbook](add-query-hook.md), and [testing strategy](../../testing/strategy.md).
