# Playbook: add an external provider

1. Add a provider interface and normalized internal DTO in External Integrations Service; never leak provider credentials or raw response shape upstream.
2. Keep mock mode as the default. Configure a real provider only through environment variables and ensure CI uses mocks/`httptest`.
3. Set explicit timeout, bounded retry/fallback, cache key/TTL, rate-limit and daily-quota behavior. Decide whether fallback is visible in response metadata.
4. Validate/normalize external data, including money/currency/date/source confidence, and return estimates rather than booking claims.
5. Add provider status/quota metrics and safe diagnostics; redact tokens and URLs containing credentials.
6. Test success, malformed payload, timeout, cache, fallback, quota/rate limit, and disabled/missing key paths.
7. Document new env keys in `infra/.env.example`, service README, [environment guide](../environment.md), and [provider quota runbook](../../operations/runbooks/provider-quota-exceeded.md).
