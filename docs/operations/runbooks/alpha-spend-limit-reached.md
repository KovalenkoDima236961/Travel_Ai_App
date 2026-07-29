# Alpha spend limit reached

Use when OpenAI daily/monthly spend ratio, spend-limit script, or provider billing indicates the alpha cap is near or exceeded.

## Triage

1. Preserve provider billing currency and note any UAH conversion as estimated.
2. Run `scripts/ai/check-openai-spend-limit.sh` with the managed alpha environment.
3. Check token usage, request count, fallback rate, and active generation jobs.
4. Confirm no normal CI job used real OpenAI.

## Recovery

1. Disable `ai_generation_enabled` or switch to fallback.
2. Keep trip creation, editing, public sharing, and settings available.
3. Cancel queued jobs if continuing them would exceed the cap.
4. Notify support of the safe user message and retry window.
5. Record the cap, current estimate, owner, and decision in the launch decision or incident record.

## Do not

- Do not raise spend caps without a reviewer and updated invite limit.
- Do not rely on UAH estimates when provider billing currency data is unavailable.
- Do not upload billing exports containing account-sensitive data.

## Escalate

Escalate for any cap exceedance, unexplained token spike, or suspected leaked key.
