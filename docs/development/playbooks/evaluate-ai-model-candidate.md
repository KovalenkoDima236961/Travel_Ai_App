# Evaluate AI Model Candidate

1. Verify the candidate adapter checksum with `scripts/ai/inspect-adapter.sh`.
2. Run `scripts/ai/evaluate-model-variants.sh --experiment-key <key> --split validation`.
3. Replace placeholder candidate metrics with measured candidate metrics when available.
4. Run `scripts/ai/check-promotion-gates.sh --metrics <metrics-json>`.
5. If all gates pass, request manual staging approval.
6. If any gate fails, record a reject, needs-more-data, or needs-retraining decision.

Never evaluate by visually sampling only. Keep validation/test/holdout split
roles separate.
