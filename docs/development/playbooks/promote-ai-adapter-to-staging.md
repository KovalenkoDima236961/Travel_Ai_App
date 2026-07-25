# Promote AI Adapter To Staging

1. Confirm validation gates passed and no hard blocks apply.
2. Copy the adapter under the approved staging `AI_MODEL_ARTIFACT_DIR`.
3. Set `AI_MODEL_VARIANT=adapter`.
4. Set `AI_ADAPTER_KEY`, `AI_ADAPTER_PATH`, `AI_ADAPTER_CHECKSUM`,
   `AI_ADAPTER_EXPERIMENT_KEY`, and `AI_ADAPTER_DATASET_VERSION`.
5. Set `AI_ADAPTER_STAGING_ENABLED=true` and `AI_ADAPTER_ENABLED=true`.
6. Keep `AI_ADAPTER_INFERENCE_ENABLED=false` unless this is a production approval.
7. Restart AI Planning Service and verify `/ready` reports adapter `ok`.
8. Monitor generation quality, latency, cost, and fallback indicators.
