# Respond To AI Model Regression

1. Disable the adapter by setting `AI_ADAPTER_ENABLED=false` or
   `AI_MODEL_VARIANT=grounded_baseline`.
2. Restart AI Planning Service and verify `/ready`.
3. Preserve current evaluation reports and runtime metadata.
4. Re-run variant evaluation on validation and test splits.
5. Record a reject or needs-retraining decision against the experiment.
6. Add a golden case or dataset quality rule covering the regression before retraining.
