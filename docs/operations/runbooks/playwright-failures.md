# Runbook: Playwright failures

1. Run the same deterministic entrypoint: `./scripts/test-frontend-e2e.sh`.
   It manages the isolated test stack unless `TEST_STACK_MANAGED=false`.
2. Inspect Playwright report, trace, video, screenshot, and CI artifact. Keep
   the stack with `KEEP_TEST_STACK=true` and inspect `docker compose -p
   travel-ai-test ... logs`.
3. Check test-stack readiness, published test ports, Chromium installation
   (`cd apps/web && npx playwright install chromium`), deterministic mock mode,
   and unique test identities.
4. Fix the synchronization/selectors: prefer role/label selectors and observable
   waits; remove arbitrary sleeps. Do not hide cross-service failure with a
   broad retry.
5. If persisted test state is stale, run `./scripts/test-stack-reset.sh`; it
   refuses non-test projects. Re-run the focused test and then the full script.
