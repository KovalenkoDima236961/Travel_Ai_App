# Running tests

## Prerequisites

- Node.js 22 and npm.
- Go version declared by each service `go.mod`.
- Python 3.12 with `services/ai-planning-service/requirements-dev.txt` installed.
- Docker Engine and Docker Compose v2 for integration, migrations, smoke, and Playwright.
- Chromium installed through Playwright: `cd apps/web && npx playwright install chromium`.

No real provider, SMTP, push, or Ollama credentials are needed.

## Fast local loops

Frontend lint, typecheck, and Vitest:

```bash
./scripts/test-frontend.sh
```

From `apps/web`, narrower commands are `npm test`, `npm run test:watch`, `npm run test:coverage`, and `npm run test -- path/to/file.test.ts`.

All Go modules:

```bash
./scripts/test-go.sh
GO_TEST_RACE=true ./scripts/test-go.sh
```

One Go service can still use `go test ./...` or its `make test` target.

Python AI service:

```bash
./scripts/test-python.sh
```

## Isolated test stack

The checked-in `infra/.env.test.example` is intentionally executable as the default test environment. The stack uses Compose project `travel-ai-test`, dedicated host ports, project-scoped volumes, mock AI/providers, and disabled delivery channels.

```bash
./scripts/test-stack-up.sh
./scripts/test-stack-down.sh
```

Keep it running for investigation with `KEEP_TEST_STACK=true`. Override the file or project only with test-scoped values:

```bash
TEST_ENV_FILE=/absolute/path/to/.env.test \
TEST_COMPOSE_PROJECT_NAME=travel-ai-test-my-branch \
./scripts/test-stack-up.sh
```

`test-stack-reset.sh` deletes and recreates only volumes belonging to a project whose name starts with `travel-ai-test`, and refuses to run unless the environment explicitly contains `APP_ENV=test`:

```bash
./scripts/test-stack-reset.sh
```

The removed test database/queue contents are not recoverable. Development and production Compose projects are outside the command's scope.

## Backend integration and migrations

```bash
./scripts/test-backend-integration.sh
```

This starts isolated Postgres, applies every service migration through `infra/Dockerfile.migrations`, and opts the existing Postgres-backed provider quota test into the test database. Other DB integration packages should follow the same opt-in environment pattern.

## Playwright

Run the deterministic full stack and Chromium suite together:

```bash
./scripts/test-frontend-e2e.sh
```

To inspect the stack after a failure:

```bash
KEEP_TEST_STACK=true ./scripts/test-frontend-e2e.sh
cd apps/web
npm run test:e2e:ui
```

When a stack is already running, set `TEST_STACK_MANAGED=false`. The default browser URL is `http://127.0.0.1:13000` in the test script; raw `npm run test:e2e` defaults to `http://127.0.0.1:3000` and honors `PLAYWRIGHT_BASE_URL`.

## Everything

```bash
./scripts/test-all.sh
```

This fails fast through frontend, Go, Python, database integration, and Playwright. It is intentionally more expensive than the normal inner loop.

## Smoke

For the ordinary core stack, run `./scripts/smoke-test.sh --core`. CI points the same script at the isolated test ports and verifies health/readiness, authenticated user/trip operations, notification unread state, provider mocks, and worker readiness.

## Troubleshooting

| Symptom | Resolution |
| --- | --- |
| Playwright says stack is not ready | Run `./scripts/test-stack-up.sh`; inspect `docker compose -p travel-ai-test ... logs`. |
| Host port is occupied | Change published ports in a copied test env and update `PLAYWRIGHT_BASE_URL`, `E2E_AUTH_URL`, and `E2E_TRIP_URL`. |
| Integration test skips | Use `./scripts/test-backend-integration.sh`; it sets `EIS_TEST_DATABASE_URL` only after verifying `APP_ENV=test`. |
| Browser binary missing | Run `cd apps/web && npx playwright install chromium`. |
| MSW reports an unhandled request | Add a behavior-specific handler or explicitly mock the request. Unhandled network is a test failure by design. |
| Stale E2E data | Run `./scripts/test-stack-reset.sh`; do not point cleanup at another Compose project. |
| CI-only failure | Run the exact script named in the job, set `CI=true` where relevant, and inspect uploaded traces/reports. |

Avoid “fixing” timing failures with sleeps. Wait for a visible state, HTTP response, queue state, or persisted revision with a bounded timeout.
