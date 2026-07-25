# Local Fine-Tuning Experiment v1

This is the v1 scaffold for one narrow task: `grounded_itinerary_generation`.
It is local-only, adapter-only, and manually promoted. It must not train on raw
production prompts, unconsented user data, comments, calendar details, receipts,
OCR, private notes, hidden system instructions, provider payloads, or unlicensed
provider text.

## Scope

- Task: `grounded_itinerary_generation`.
- Method: LoRA or QLoRA adapter training against a Hugging Face-compatible base.
- Dataset source: Trip Service curated dataset exports only.
- Runtime default: adapters disabled; base/mock/Ollama behavior remains unchanged.
- Promotion: manual staging/production decision after validation and test reports.

The current local inference default is `OLLAMA_MODEL=llama3.1:8b`. Ollama/GGUF
runtime artifacts are not directly PEFT-trainable. For training, choose and
record a Hugging Face-compatible base model name and exact revision, then record
how it maps to the local Ollama runtime family before evaluating an adapter.

## Required Controls

- `AI_FINE_TUNING_EXPERIMENTS_ENABLED=false` by default.
- `AI_ADAPTER_INFERENCE_ENABLED=false` by default.
- `AI_ADAPTER_STAGING_ENABLED=false` by default.
- `AI_ADAPTER_ENABLED=false` by default.
- Holdout examples are never loaded into the training bundle.
- Dataset checksums, task distribution, split disjointness, and private-data
  markers are validated before training.
- Runtime readiness rejects missing/outside-artifact/checksum-mismatched
  adapters unless explicit non-strict base fallback is configured.
- `/version`, `/ready`, and AI response metadata may include model variant,
  adapter key, dataset version, experiment key, and checksum, but never adapter
  filesystem paths, raw prompts, raw responses, or user/trip identifiers.

## Dataset Export

Trip Service exports live under `AI_DATASET_EXPORT_DIR` and contain:

- `train.jsonl`
- `validation.jsonl`
- `test.jsonl`
- `holdout.jsonl`
- `manifest.json`
- `checksums.txt`
- `README.md`

Validate first:

```bash
scripts/ai/validate-dataset-export.sh data/ai-datasets/<export-dir>
scripts/ai/validate-training-run.sh --config services/ai-planning-service/training/example.config.json --dataset-path data/ai-datasets/<export-dir>
```

The training CLI scans every split for private markers. It loads train,
validation, and test metadata for evaluation, but never uses holdout examples
for training.

## Training

Install training dependencies only on the local training host:

```bash
cd services/ai-planning-service
python3 -m pip install -r requirements-training.txt
AI_FINE_TUNING_EXPERIMENTS_ENABLED=true python3 -m training.cli --config training/example.config.json
```

Or use the optional Compose profile:

```bash
docker compose -f infra/docker-compose.yml --profile ai-training run --rm ai-training-runner \
  python -m training.cli --config /artifacts/config.json --dataset-path /datasets/<export-dir>
```

CI must use `--dry-run` or `--cpu-smoke` only. Full training is intentionally
not part of pull-request validation.

## Evaluation

Generate a variant comparison report:

```bash
scripts/ai/evaluate-model-variants.sh --experiment-key <experiment-key> --split validation
scripts/ai/check-promotion-gates.sh --metrics evals/ai-itinerary/reports/experiments/<experiment-key>/validation-comparison.json
```

Required variants:

- `base`
- `grounded_baseline`
- `fine_tuned_candidate`

Promotion gates require schema validity, grounding/citation quality, no private
data, no holdout leakage, and bounded cost/latency regression.

## Runtime Activation

Adapter activation requires all of the following:

- The adapter artifact is under `AI_MODEL_ARTIFACT_DIR`.
- `AI_ADAPTER_KEY`, `AI_ADAPTER_PATH`, and strict-env `AI_ADAPTER_CHECKSUM` are set.
- `AI_MODEL_VARIANT=adapter`.
- `AI_ADAPTER_ENABLED=true`.
- `AI_ADAPTER_INFERENCE_ENABLED=true`, or `AI_ADAPTER_STAGING_ENABLED=true` in
  local/staging only.

Production rejects `AI_ADAPTER_STAGING_ENABLED=true` and requires the inference
gate plus checksum verification.

## Non-Goals

- No training on broad chat/copywriting/general assistant tasks.
- No automatic promotion.
- No hosted training pipeline.
- No training directly against raw Ollama/GGUF artifacts.
- No public download endpoint for adapters or dataset exports.
