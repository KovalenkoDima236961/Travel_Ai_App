# Runbook: provider quota exceeded

1. Identify provider/category from External Integrations logs, protected
   `/ops/providers/status` or `/ops/providers/quotas` (when enabled), and
   Prometheus/Grafana provider metrics.
2. Confirm whether this is per-minute rate limiting, daily quota, missing key,
   upstream outage, or a local mock configuration error. Record provider,
   endpoint, timestamp, configured fallback, and request/correlation ID.
3. Verify the configured mock/fallback behavior. UI must say estimate/fallback
   rather than claim fresh availability, booking, price, or schedule data.
4. In local development only, the protected `reset-dev` quota endpoint may be
   used after confirming it is a dev environment. Do not reset production usage
   counters to mask a provider limit.
5. For staging/production, reduce traffic/cache appropriately, wait for reset
   window or provider approval, and communicate degraded user behavior. Do not
   add a real provider call in CI as a diagnostic.
