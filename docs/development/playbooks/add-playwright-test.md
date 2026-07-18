# Playbook: add a Playwright test

1. Put browser tests in the existing `apps/web` Playwright structure and use its auth/API setup helpers and deterministic mock test stack.
2. Create data through supported API helpers where possible; use unique test identity/date data and clean through the test-stack lifecycle.
3. Prefer role, label, and visible-name selectors. Use `data-testid` only when no accessible stable selector describes the behavior.
4. Assert user-observable state, not implementation timing. Wait for navigation/network/UI state; avoid fixed sleeps and real provider/AI calls.
5. Keep the scenario focused on a cross-service seam; validation matrices belong in unit/component/handler tests.
6. Run `./scripts/test-frontend-e2e.sh`; use `KEEP_TEST_STACK=true` to inspect failures and keep trace/video/screenshot artifacts useful.

See [Playwright failures](../../operations/runbooks/playwright-failures.md).
