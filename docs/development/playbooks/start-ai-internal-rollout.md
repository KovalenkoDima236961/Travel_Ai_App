# Start AI Internal Rollout

1. Confirm shadow metrics have sufficient samples and no critical guardrails.
2. Enable backend flag `ai_candidate_internal_rollout_enabled`.
3. Set deployment `status='internal'` and `traffic_mode='internal'`.
4. Verify only ops/internal users receive `Experimental AI`.
5. Record explicit feedback and switch-back reasons.
