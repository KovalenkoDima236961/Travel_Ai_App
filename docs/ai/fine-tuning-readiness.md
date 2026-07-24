# Fine-tuning readiness

V1 does not fine-tune, upload, or train any model. It adds the governance, storage, sanitizer, scoring, review, versioning, and export gates needed before a future controlled experiment can be considered.

## Non-goals

- No model training job.
- No provider upload.
- No raw production-trip dump.
- No training on RAG holdout cases.
- No replacement for provider-backed verification or grounding.

RAG/provider verification remains the factual source for travel knowledge. A trained model may learn style, structure, repair behavior, and preference handling, but current provider facts must still come from the grounding and verification layer.

## Dataset lifecycle

1. Candidate extraction creates records in `ai_training_examples` from manual examples, golden eval cases, or future consented user-derived events.
2. Sanitization removes known sensitive fields and redacts email, phone, token, share URL, hidden prompt, calendar, receipt/OCR, comments, private notes, raw logs, and internal IDs.
3. Scoring assigns a 0-1 quality score from schema validity, grounding quality, itinerary validity, place validity, schedule plausibility, preference match, budget plausibility, acceptance signal, and privacy confidence.
4. Human review approves or rejects examples. Approval is blocked when consent is missing/revoked, sanitization failed, quality is too low, private data is detected, or provider license is not allowed.
5. Version building selects approved, sanitized, consent-valid examples, deduplicates by source group, assigns deterministic train/validation/test/holdout splits, and writes a manifest checksum.
6. Export writes a private JSONL package only when `AI_DATASET_EXPORT_ENABLED=true`.

## Readiness gates

`GET /ops/ai/fine-tuning/readiness` reports `ready=false` until these baseline gates pass:

- At least 500 approved high-quality examples.
- No unresolved sanitization failures.
- At least one frozen holdout example.
- No duplicate groups requiring review.
- Baseline eval status acknowledged as required before training.

Before any future LoRA or adapter experiment, also require a fresh benchmark report, a rollback plan to the current base model, a model-card update, privacy/security review, and human spot checks.

## Provider data

Provider-derived knowledge records are subject to the same licensing rules as any other candidate training data, and they are stricter than they look: a record's license may permit this application to use facts for planning without permitting model training on provider text.

For dataset work, reference provider facts through grounding IDs rather than copying provider text. Exclude records whose source license is unknown, incompatible, or not allowed. Records with `review_status` of `rejected` or `merged`, and records below the weak-grounding quality floor, are not eligible candidates.

Mock provider fixtures are synthetic and clearly labelled. They are usable for pipeline testing and must never be exported as if they were real travel data. See [trusted travel data providers](trusted-travel-data-providers.md).

## Local commands

```bash
scripts/ai/validate-training-examples.sh
scripts/ai/build-synthetic-dataset.sh
ALLOW_NOT_READY=1 scripts/ai/run-fine-tuning-readiness-check.sh
scripts/ai/validate-dataset-export.sh data/ai-datasets/<export-dir>
```

## Configuration

Trip Service owns these settings:

- `AI_DATASET_EXPORT_ENABLED=false`
- `AI_DATASET_EXPORT_DIR=./data/ai-datasets`
- `AI_DATASET_EXPORT_RETENTION_DAYS=30`
- `AI_DATASET_MIN_AUTO_REVIEW_SCORE=0.70`
- `AI_DATASET_MIN_APPROVAL_SCORE=0.85`
- `AI_DATASET_REQUIRE_HUMAN_REVIEW=true`
- `AI_DATASET_GOLDEN_CASES_DIR=./evals/ai-itinerary/cases`
- `AI_DATASET_MANUAL_EXAMPLES_DIR=./data/ai-training/manual`

Exports are private filesystem packages. They are never public URLs.
