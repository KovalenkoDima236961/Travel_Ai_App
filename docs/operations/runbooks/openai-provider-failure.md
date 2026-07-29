# OpenAI Provider Failure

Use this runbook when AI Planning Service reports OpenAI errors, high latency,
rate limits, or fallback spikes.

## Triage

1. Check `/ready`; it must not call OpenAI and must not expose secrets.
2. Check `/metrics` for:
   - `ai_provider_requests_total`
   - `ai_provider_request_duration_seconds`
   - `ai_provider_errors_total`
   - `ai_provider_fallbacks_total`
3. Confirm `OPENAI_ENABLED=true`, provider selection, fallback selection, and
   operation model aliases in the runtime environment.
4. Inspect logs for safe error codes only. Do not request raw prompts or raw
   provider response bodies.

## Common Actions

- Authentication or permission errors: correct secret/configuration and restart
  the affected AI Planning Service instances.
- Sustained 429 or quota errors: lower traffic, switch to fallback, or raise the
  provider quota through the account owner.
- Timeout or 5xx errors: keep fallback enabled and watch fallback quality flags.
- Invalid structured output: verify the configured model supports Structured
  Outputs and run alpha evals before retrying rollout.

## Rollback

Set:

```bash
AI_MODEL_PROVIDER=mock
ITINERARY_GENERATOR_MODE=mock
COPILOT_MODE=mock
TRIP_RECAP_AI_MODE=mock
OPENAI_ENABLED=false
```

For local Ollama fallback:

```bash
AI_MODEL_PROVIDER=ollama
AI_MODEL_PROVIDER_FALLBACK=mock
```

Never edit or rotate the OpenAI API key through the UI.

