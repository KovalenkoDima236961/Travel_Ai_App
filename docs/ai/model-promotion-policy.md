# AI Model Promotion Policy

Fine-tuned adapters can move to staging or production only through a manual
promotion decision. A completed training job is never enough.

## Minimum Evidence

- Frozen/exported curated dataset version with checksums.
- Training experiment manifest with base model name, revision, method, seed,
  dataset version, adapter checksum, and hardware notes.
- Validation and test comparison reports for `base`, `grounded_baseline`, and
  `fine_tuned_candidate`.
- Promotion gate output from `scripts/ai/check-promotion-gates.sh`.
- Reviewer reason recorded for approve/reject/retrain decisions.

## Hard Blocks

- Holdout examples used in training.
- Private data, raw prompts, raw provider payloads, comments, receipt/OCR,
  calendar details, user IDs, trip IDs, or secrets in the dataset or artifacts.
- Missing or mismatched adapter checksum.
- Unrecorded base model license/revision.
- Any production adapter activation without `AI_ADAPTER_INFERENCE_ENABLED=true`.
- Any automatic promotion path.

## Promotion Gates

Default gates:

- Schema valid rate: at least 0.99.
- Factual precision: at least 0.92.
- Grounding citation rate: at least 0.95.
- No-PII rate: 1.0.
- Cost regression: at most 10%.
- Latency regression: at most 25%.
- Holdout used for training: false.

Staging approval may be granted after validation gates pass. Production approval
requires staging observation, test split results, and an explicit production
review. Holdout reports are diagnostic only and must not influence training.

## Rollback

Rollback is configuration-only: unset `AI_ADAPTER_ENABLED` or restore
`AI_MODEL_VARIANT=grounded_baseline`, then restart the AI Planning Service.
Preserve the adapter artifact and experiment records for audit until retention
policy allows cleanup.
