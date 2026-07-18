# CI quality gates

GitHub Actions workflow `.github/workflows/ci.yml` runs the same tools available locally.

## Jobs

| Job | Gate |
| --- | --- |
| Compose config | Validates safe CI/test environments and every Compose profile. |
| Frontend | ESLint, TypeScript, Vitest coverage, and production Next build. |
| Go tests | `go test ./...` for each of six Go services with module caching. |
| AI Planning tests and lint | Ruff and pytest on Python 3.12 in deterministic mode. |
| Backend integration | Isolated Postgres, all migrations, and opted-in repository integration tests. |
| Fresh migrations | Applies all service migrations to a new database through the deployment migration image. |
| Playwright E2E | Builds/starts the mock full stack and runs Chromium critical flows. |
| Docker builds | Builds every application and migration image without pushing. |
| Full-stack smoke | Checks core readiness plus authenticated cross-service operations. |
| Security jobs | Gitleaks, gosec, govulncheck, npm audit, Bandit, pip-audit, Semgrep, and Trivy according to `docs/security/tools.md`. |

Race tests and additional Playwright browsers are candidates for scheduled/nightly jobs; they are not required on every pull request in v1.

## Recommended required checks

Protect the default branch with these required checks:

- Compose config
- Frontend
- Go tests for every matrix service
- AI Planning tests and lint
- Backend integration
- Fresh migrations
- Playwright E2E
- Full-stack smoke
- all security jobs

Docker builds should also be required when runner time is acceptable; otherwise require them for release branches and changes touching Dockerfiles, dependency locks, or Compose.

## Artifacts

Frontend coverage is uploaded on every run. Playwright HTML, JUnit, screenshots, video, and traces are uploaded on failure. Stack logs are printed on integration/E2E/smoke failure. Artifacts must never include `.env` files, tokens, raw receipt OCR, model prompts, private request bodies, or provider responses containing credentials.

## Debugging a failure

1. Open the first failing job rather than downstream cancellations.
2. Run the exact local command documented in the step or in [running-tests.md](running-tests.md).
3. For Playwright, open the trace from `apps/web/test-results` or the HTML report.
4. For stack failures, inspect the failed service's readiness and preceding logs; do not increase timeouts before identifying the dependency.
5. For migrations, reproduce against a fresh test project/volume.
6. For security, follow the severity and exception policy in `docs/security/tools.md`; do not silently suppress findings.

## Coverage and snapshots

CI reports coverage but has no arbitrary global percentage threshold in v1. Changed critical logic should add focused behavior coverage. Snapshot-heavy testing is discouraged; there is no visual-regression baseline. If a small intentional snapshot is introduced later, review its semantic change rather than updating it blindly.

## Path filtering

V1 deliberately runs the broad safety net on pull requests. This repository has cross-service DTO, Compose, migration, and shared security coupling, so premature path filtering risks missing regressions. Add path-aware optimization only after job timing data shows a material benefit and dependency paths are documented.
