# OpenAI alpha checklist

Use this before any real OpenAI alpha run. Normal CI and default rehearsals stay mock/fallback-only.

## Configuration

- [ ] `OPENAI_ENABLED=true` only in the managed alpha secret environment.
- [ ] `AI_MODEL_PROVIDER=openai` and `AI_ITINERARY_GENERATOR_MODE=openai`.
- [ ] `AI_MODEL_PROVIDER_FALLBACK=mock` or `ollama`.
- [ ] `AI_MODEL_PROVIDER_FALLBACK_ENABLED=true`.
- [ ] Operation model aliases are configured: default, itinerary, regeneration, repair, budget, checklist, and Copilot if enabled.
- [ ] `OPENAI_STORE_RESPONSES=false`.
- [ ] `AI_PROMPT_LOGGING_ENABLED=false`.
- [ ] `AI_OBSERVABILITY_REDACTION_ENABLED=true`.
- [ ] Redacted prompt/response storage is disabled unless separately approved.

## Limits

- [ ] Per-user daily generation limit configured.
- [ ] Per-trip daily generation limit configured.
- [ ] Global concurrent OpenAI request limit configured.
- [ ] Daily and monthly spend limits configured in UAH for app-local budgeting.
- [ ] Provider billing currency is preserved separately from estimated UAH.
- [ ] Input tokens, context bytes, grounding places/documents, retries, and timeouts are bounded.
- [ ] Repair is enabled with `OPENAI_MAX_REPAIR_ATTEMPTS=1`.

## Privacy

- [ ] API key is server-side only and not embedded in Next.js build args.
- [ ] No committed real key, old key, raw prompt, raw response, or user itinerary in reports.
- [ ] Prompt sanitizer blocks raw OCR, comments, notes, emails, tokens, and private calendar details.
- [ ] AI trace detail pages expose safe summaries only to Ops admins.
- [ ] Evaluation artifacts are redacted and safe to upload.

## Fallback and kill switches

- [ ] Ops can disable `ai_generation_enabled`.
- [ ] Ops can disable `ai_repair_enabled`.
- [ ] Ops can disable `public_sharing_enabled` and `data_exports_enabled`.
- [ ] Fallback path is tested with `scripts/ai/run-alpha-provider-evals.sh --mock`.
- [ ] Limited real eval uses `--openai --allow-real-openai --max-requests <1..5>` and protected approval.

## Alerts

Prometheus rules live in `infra/observability/prometheus/rules/alpha-openai-alerts.yml`.

- [ ] Daily spend near threshold.
- [ ] Monthly spend near threshold.
- [ ] Unexpected token spike.
- [ ] High fallback rate.
- [ ] High repair rate.
- [ ] Quota/rate-limit/budget/concurrency errors.
- [ ] Provider auth/permission failure.
- [ ] Worker queue delay and generation failure spike.
- [ ] External provider circuit open.

Spend-ratio alerts require an exported `openai_spend_daily_ratio` and `openai_spend_monthly_ratio` metric. Until that metric exists, run `scripts/ai/check-openai-spend-limit.sh` as release evidence and document the result in the decision record.

## Key rotation rehearsal

1. Record the rotation window, owner, and request ID. Do not paste the old key.
2. Replace the managed secret value.
3. Restart/reload the AI Planning and Trip paths that read it.
4. Verify `/ready`, one mock fallback eval, and one approved capped OpenAI eval if needed.
5. Confirm no old key or raw provider error appears in logs.
6. Revoke the old key in the OpenAI console.
7. Attach only safe command outputs and version data to the launch decision.
