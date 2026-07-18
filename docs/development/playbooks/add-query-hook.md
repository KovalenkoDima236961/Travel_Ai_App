# Playbook: add a TanStack Query hook

1. Keep request code in `apps/web/src/lib/api/<domain>.ts`; normalize errors through existing client helpers.
2. Add a stable, parameterized key in `src/lib/query-keys.ts`. Put private trip data under the trip-detail prefix and include every server-affecting parameter.
3. Do not run until required ID/token/permission state exists. Use the project defaults (normally 30 seconds); poll only queued/running work and stop on terminal state.
4. On mutation, invalidate the smallest affected subtree—e.g., expense, budget, health, activity—not every trip query.
5. Surface recoverable errors and preserve server conflict information. Never silently retry an itinerary revision conflict.
6. Ensure auth is provided by existing fetch/auth helpers; do not place secrets in query keys, logs, or browser storage.
7. Add MSW handler and typed fixture in `apps/web/test`, then test request shape, success, error, disabled, and invalidation behavior.

Run `./scripts/test-frontend.sh`. See [API overview](../../api/overview.md) and the Web README's query conventions.
