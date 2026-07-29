# AI Model Rollout Regression Runbook

Use this when a candidate model deployment shows quality, latency, reliability,
privacy, or load regression.

## Identify Deployment

1. Query the known deployment with
   `scripts/ai/check-online-model-metrics.sh --deployment-id <id>` or
   `GET /ops/ai/model-deployments/{deploymentId}/online-summary`.
2. Record deployment key, environment, task type, traffic mode, rollout percent,
   guardrail status, and latest deployment event.
3. Confirm the grounded baseline deployment is available.

## Inspect Guardrails

Review the latest rollout window:

- sample count and skipped-shadow count
- candidate failure and parse-failure rate
- hallucination and destination mismatch deltas
- repair-rate increase
- p50/p95 latency delta
- language-specific regressions
- capacity/load indicators

Do not inspect raw prompts or full generated output unless a local/staging-only,
encrypted, audited debug retention mode is explicitly enabled.

## Pause Rollout

For user-visible candidate traffic, immediately pause or retire the deployment
with `scripts/ai/pause-model-deployment.sh --deployment-id <id> --reason
"<reason>"` or `scripts/ai/rollback-ai-model.sh --deployment-id <id> --reason
"<reason>"`. Automatic critical guardrail failures should already create a
`guardrail_paused` event. If not, create a manual `paused` or `rollback` event.

## Verify Baseline Routing

1. Confirm new assignments are `baseline_only`.
2. Confirm user-visible generation badges show `Standard AI`.
3. Confirm no new candidate itinerary versions are created.
4. Confirm shadow queue depth is draining or paused.

## Preserve Safe Evidence

Keep comparison rows, aggregate rollout windows, deployment events, error
codes, and redacted ops notes. Do not export prompts, complete itineraries, or
private user context.

## Decide Outcome

Choose one:

- Resume shadow only after infrastructure recovery.
- Keep paused and request candidate fix.
- Reject candidate.
- Retire adapter after audit retention allows.
- Retrain from approved curated data only.

## Communication

Normal users do not need model details. If user-visible candidate output caused
visible degradation, communicate that experimental AI was rolled back to
Standard AI and no existing itinerary was changed automatically.
