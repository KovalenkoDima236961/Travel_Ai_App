# Start AI Shadow Evaluation

1. Confirm offline validation/test gates passed.
2. Confirm `ai_shadow_evaluation_enabled` and `AI_SHADOW_ENABLED` are enabled
   only in the target environment.
3. Run `scripts/ai/enable-shadow-rollout.sh --deployment-id <id>
   --sample-percent <small-number> --reason "<reason>"`.
   Trip Service sets `status='shadow'`, `traffic_mode='shadow'`, and
   `shadow_sample_percent`, then records `enabled_shadow` with the reason.
5. Watch shadow queue depth, comparison completion rate, skipped-shadow reasons,
   and guardrail windows.
