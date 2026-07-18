# Testing strategy

## Purpose

Testing & CI/CD Quality Gates v1 is the change-safety baseline for Travel AI App. It favors deterministic business-rule tests and service-level HTTP tests, then uses a deliberately small Playwright suite for cross-service confidence. It does not pursue 100% coverage or turn every edge case into a browser test.

## Test pyramid

| Layer | Expected volume | Scope | Typical command |
| --- | ---: | --- | --- |
| Type, lint, schema, and pure unit tests | Many | TypeScript utilities, Go domain/services, Python schemas, prompt builders, parsers, validators | `./scripts/test-frontend.sh`, `./scripts/test-go.sh`, `./scripts/test-python.sh` |
| Component and handler tests | Many | Forms, permissions, offline behavior, `httptest`, FastAPI endpoints, mock providers | Same service-level commands |
| Database and queue integration tests | Solid but focused | Migrations, Postgres repositories, provider quota store, service authorization boundaries, worker transitions | `./scripts/test-backend-integration.sh` |
| Full-stack browser E2E | Small | Authentication, trip creation, mock generation start, list/detail, public sharing, accessibility smoke | `./scripts/test-frontend-e2e.sh` |
| Stack smoke | Very small | Health/readiness, auth, trip CRUD, notifications, provider mocks, worker readiness | `./scripts/smoke-test.sh --core` |

Unit and component tests should provide precise failure messages for business rules. Integration tests prove wiring and persistence behavior. Playwright proves only the highest-value seams that cannot be trusted from one service in isolation.

## Frontend unit and component scope

Vitest is the only frontend unit runner. Pure utilities continue to run in the Node environment. Files that exercise browser behavior use the `jsdom` Vitest environment and the shared setup in `apps/web/test/setup.ts`.

The suite owns:

- Zod and form behavior for auth, trip creation, profile/preferences, budget, expenses, receipts, itinerary editing, route inputs, and notification settings.
- Itinerary normalization, lookup, ordering, cross-day moves, diff/merge/conflict handling, quality analysis, instruction building, and display formatting.
- Budget/category/day totals, missing estimates, currency formatting, cost splits, settlements, and confidence display helpers.
- Haversine and fallback estimates, route metrics, validation, warnings, labels, and map data preparation.
- IndexedDB cache separation, summary/detail storage, mutation coalescing, sync conflicts, logout cleanup, and stale cleanup.
- Owner/editor/viewer/public/workspace/ops visibility rules.
- Notification list/unread/preference behavior and SSE parsing/navigation updates.
- Focused Testing Library and `jest-axe` checks on critical forms and dialogs.

MSW handlers and typed fixtures live under `apps/web/test`. API client tests still assert exact request behavior; critical DTO fixtures provide a lightweight contract seam without introducing a contract platform.

## Playwright E2E scope

Chromium is required on every pull request. Firefox and WebKit can be scheduled later if their runtime stays affordable. V1 browser coverage is intentionally limited to:

- register, login, logout, invalid credentials, and protected-route redirect;
- create a trip through the stepper, start deterministic mock generation, open trip detail, and find the trip again in the list;
- create a public share through an authenticated API helper and verify the anonymous page is sanitized and read-only;
- serious/critical accessibility smoke on landing and auth pages.

The fixture layer provides deterministic credentials, direct API setup, auth storage state, fixed dates, and mock-only service URLs. UI selectors prefer roles, labels, and visible names. `data-testid` should be introduced only when an accessible selector cannot describe stable user behavior.

### What not to test with E2E

Do not use Playwright for every validation branch, parser failure, calculation, retry transition, provider mapping, locale string, merge combination, or permission matrix cell. Those belong in unit, component, handler, or integration tests. Do not add screenshot baselines in v1. Failure screenshots, videos, and traces are sufficient.

## Go test scope

Each Go module runs independently with `go test ./...`. Package-local fakes, fixed clocks, deterministic IDs, `httptest`, and repository interfaces remain the preferred unit style.

Critical ownership includes:

- Auth: password hashing, validation, access/refresh token validation, refresh hashing/rotation/revocation, logout, and rate limits.
- User: profile/preferences validation, language/currency constraints, workspace membership and invitation rules, internal lookup authorization.
- Trip: trip and itinerary validation, revision conflicts, access resolution, budgets, routes, health/verification, policy and approval rules, repairs, checklist/reminders, splits/settlements, public/export/AI sanitization, and generation lifecycle.
- Notification: channel/category preferences, unread state, grouping/deduplication/digests, quiet hours/mutes, SSE non-blocking behavior, mock delivery, and self-notification suppression.
- Worker: claim/retry/backoff/DLQ/cancellation/idempotency, correlation propagation, generation and due-reminder processing.
- External integrations: normalized provider DTOs, deterministic mocks, timeouts/fallbacks/caches/quotas, routes/weather/places/prices/availability/transport, calendar crypto and free/busy privacy.

`GO_TEST_RACE=true ./scripts/test-go.sh` enables race detection for local or scheduled runs. Pull requests use the faster non-race command; package Makefiles retain race-enabled `make test` where configured.

## Backend integration scope

The `test` Compose profile creates an isolated Compose project and volumes. Migrations are applied from the same migration image used by deployment. Integration tests must opt in with test-only environment variables; destructive cleanup requires both an `APP_ENV=test` file and a Compose project beginning with `travel-ai-test`.

Repository/HTTP integration priorities are authentication rotation, trip ownership and stale revisions, viewer denial, public sanitization, notification internal-token enforcement and preferences, deterministic provider endpoints/fallbacks/quotas, and worker claim/retry/idempotency. V1 runs the existing Postgres-backed provider quota integration test and the authenticated full-stack smoke flow; new repository tests should use the same isolated database rather than a production-like URL.

## Python AI service scope

Pytest covers Pydantic validation, itinerary schemas, prompt construction, JSON extraction, repair/fallback behavior, language handling, RAG chunk selection, privacy filtering, Copilot construction, checklist/discovery/route/recap logic, and FastAPI health/readiness/generation endpoints. CI always uses mock mode and never requires Ollama, Chroma, a real model, or an external network.

## Fixtures and time

Canonical fixture identities, dates, routes, costs, notifications, and file samples are documented in [fixtures.md](fixtures.md). Fixed dates are used for business assertions. Time-sensitive code receives a fake clock or explicit timestamp. Machine timezone must not influence expectations; Playwright explicitly uses `Europe/Bratislava`, while service tests should use timezone-aware values.

Test data is isolated by user/test ID. Tests never depend on order. E2E emails are unique per run and retry. AI assertions target schema and stable fields rather than long generated prose.

## Provider and delivery strategy

CI uses mock modes for AI, places, routing, weather, exchange rates, prices, availability, calendar, email, and push. SMTP and push delivery are disabled. Provider-specific adapters are tested with local `httptest`/mock responses. Any unhandled frontend request fails in MSW-backed component tests. No CI test may require real credentials or make a real provider call.

## Critical regression map

| Regression area | Primary layer | Cross-stack proof |
| --- | --- | --- |
| Register/login/logout/refresh | Auth unit/handler/integration | Playwright auth + smoke |
| Profile/preferences/onboarding | User and component tests | Smoke user profile |
| Trip create/list/detail | Trip handler/service tests | Playwright trip + smoke |
| AI generation lifecycle | Trip/worker/Python mock tests | Playwright generation start + worker smoke |
| Itinerary edit/save/conflict | TypeScript diff/merge + Trip revision tests | Targeted browser test when UI contract changes |
| Route builder | Route utility/component + provider handler tests | Trip creation and provider smoke |
| Budget/expenses/settlements | Go business rules + frontend utility/forms | Smoke mutation coverage |
| Receipt upload | File validation/security/handler tests | Safe smoke fixture; no real OCR |
| Checklist/reminders | Trip/worker rules + components | Worker reminder integration |
| Collaborator/workspace permissions | Trip/User permission matrices + UI visibility | API-seeded browser tests as high-risk flows change |
| Public share | Sanitization/handler/component tests | Playwright anonymous share |
| Notifications | Notification service + frontend hooks/SSE | Smoke unread count |
| Approval/policy | Trip service rule tests + permission UI | Not every policy cell in E2E |
| Offline cached trip | IndexedDB/queue component tests | One minimal browser offline test when cache contract changes |
| Exports | Sanitizer/formatter tests | No browser matrix |
| Internal authorization | Middleware/handler tests | Smoke protected internal call |
| Provider mocks | External integrations tests | Smoke mock endpoints |
| Worker processing | Worker/Trip state tests | Queue readiness and smoke |

## Coverage expectations

Coverage is reported where practical and is not gated on an arbitrary repository-wide percentage in v1. Reviewers expect meaningful coverage around changed critical logic, especially frontend utilities/hooks, Go service/domain packages, and Python schema/prompt/parser code. A coverage decrease in a touched critical package needs explanation. Generated files, composition plumbing, and visual-only wrappers do not need artificial tests.

## CI quality gates

Pull requests run Compose/environment validation, frontend lint/typecheck/unit coverage/build, every Go module, Python lint/tests, Postgres integration, fresh migrations, Chromium Playwright, all Docker builds, security scanners, and a full-stack smoke job. See [ci.md](ci.md) for required checks and artifacts.

## Reliability rules

- Never use arbitrary long sleeps; poll observable state with a deadline.
- Reset/isolate persisted state and use unique identities.
- Do not depend on current time, host timezone, execution order, real randomness, external networks, or provider availability.
- Keep timeouts short at unit/integration layers and larger only at full-stack startup boundaries.
- Treat a flaky test as a defect: fix its synchronization or move the assertion to the correct layer.
- Never weaken authorization, file validation, migration safety, or provider isolation to make a test easier.

Local commands and troubleshooting are in [running-tests.md](running-tests.md).
