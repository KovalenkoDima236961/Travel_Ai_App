# Increase AI Rollout Percentage

1. Confirm candidate is approved for staged rollout and baseline rollback is
   available.
2. Enable backend flags for percentage rollout and user opt-in.
3. Set `rollout_percent` to the next small step, usually 1-5 percent.
4. Keep assignment salt unchanged unless intentionally resetting buckets.
5. Review rollout windows before each increase.
