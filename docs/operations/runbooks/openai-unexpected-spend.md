# OpenAI Unexpected Spend

Use this runbook when token usage, provider invoices, or Ops summaries indicate
unexpected OpenAI cost.

## Immediate Containment

1. Set `OPENAI_ENABLED=false` or switch provider mode to `mock`.
2. Keep `AI_MODEL_PROVIDER_FALLBACK=mock` only if lower-quality generation is
   acceptable and clearly marked for review.
3. Confirm no frontend or public-share path can trigger a private generation
   without normal authorization.
4. Capture `/metrics` output for token counters and provider request counts.

## Investigation

- Compare request volume by operation and model group.
- Check whether a rollout changed `AI_MODEL_PROVIDER`, operation model aliases,
  `OPENAI_MAX_OUTPUT_TOKENS`, or request limits.
- Verify no normal log contains raw prompts, raw outputs, or API keys.
- Check provider-side billing in the OpenAI dashboard; preserve original billing
  currency and treat UAH conversion as an estimate only.

## Follow-up

Before re-enabling OpenAI in alpha, ensure durable usage ledger and spend-limit
enforcement are active. The provider layer currently exposes metrics and config
fields, but distributed spend protection must be enforced by the owning backend
service.

