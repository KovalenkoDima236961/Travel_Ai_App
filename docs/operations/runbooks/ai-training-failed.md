# AI Training Failed

## Symptoms

- `training.cli` exits non-zero.
- Experiment status is `failed`.
- No adapter directory or checksum was produced.
- GPU host reports out-of-memory or missing training dependencies.

## Response

1. Confirm the dataset export still validates.
2. Check `experiment-manifest.json`, `metrics.jsonl`, and checkpoint logs.
3. Verify the base model name/revision/license and local model access.
4. If hardware failed, lower batch size/sequence length or switch LoRA/QLoRA method.
5. Do not promote partial adapters.
6. Mark the experiment failed or needs-retraining with the safe error summary.

## Safety Checks

- Do not paste raw prompts, user records, provider payloads, or adapter paths into public tickets.
- Do not use holdout examples to tune hyperparameters.
- Keep `AI_ADAPTER_ENABLED=false` until a complete adapter passes checksum and evaluation gates.
