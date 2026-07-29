# Pause and Roll Back AI Model

1. Pause with `scripts/ai/pause-model-deployment.sh --deployment-id <id>
   --reason "<reason>"` or retire with
   `scripts/ai/rollback-ai-model.sh --deployment-id <id> --reason "<reason>"`.
2. Trip Service sets traffic to `disabled`, clears rollout/shadow percentages,
   and records `paused` or `rollback` with the concrete reason.
3. Verify new assignments route to `grounded_baseline`.
4. Keep comparison rows and deployment events for audit.
5. Do not mutate already-created itineraries.
