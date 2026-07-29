# Alpha OpenAI outage

Use when OpenAI auth, rate limit, quota, timeout, invalid response, or availability errors affect alpha.

## Triage

1. Check `ai_provider_errors_total{provider="openai"}` by safe error code.
2. Check fallback rate, token spike alert, repair rate, and spend-limit evidence.
3. Verify AI Planning Service `/ready` and model alias configuration.
4. Confirm `OPENAI_STORE_RESPONSES=false` and prompt logging remains disabled.

## Recovery

1. For auth/permission errors, disable AI generation or switch to fallback, then rotate/fix the key.
2. For quota/budget errors, disable OpenAI and leave trip creation available.
3. For rate limits/timeouts, reduce invite pace and route to fallback if configured.
4. For invalid response/repair failures, pause real OpenAI generation and use mock fallback for new generation.
5. Run a capped eval only after recovery and approval.

## Key rotation

Follow `docs/releases/openai-alpha-checklist.md`. Never store the old key in reports.

## Escalate

Escalate immediately for auth failure, budget exceeded, provider key exposure suspicion, or fallback failure.
