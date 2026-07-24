# AI dataset governance

## Ownership

Trip Service owns dataset project metadata, consent records, candidate examples, review events, version manifests, and private exports. Worker Service may orchestrate future jobs, but it must call owning Trip Service endpoints rather than writing these tables directly.

## Consent

User-derived examples require explicit consent before approval or export. Consent can be global for future examples, scoped to a trip, or scoped to an itinerary version. Revocation marks user-derived candidates as `revoked` and blocks approval/export.

Synthetic, manual curated, and golden holdout examples use `consent_status=not_required`, but they still require provenance and license review.

## Sanitization

The sanitizer removes or redacts:

- User identifiers, emails, phone numbers, names, exact home addresses.
- Receipts, OCR, calendar details, comments, collaboration messages, private notes.
- Tokens, passwords, API keys, public share tokens, storage paths.
- Raw prompts, hidden system instructions, chain-of-thought markers, raw logs.
- Internal database IDs except explicitly allowed grounding/place identifiers.

`sanitization_status=failed` is a hard blocker. `needs_review` cannot be approved until re-sanitized or manually corrected.

## Quality and review

Quality scoring is deterministic and advisory. Human approval remains required for v1 exports. Reviewers must confirm:

- Schema-valid input/output.
- Grounding IDs or licensed facts.
- No hallucinated place claims.
- Reasonable schedule and budget.
- Preference fit.
- No private data.
- License permits this training use.

## Versioning and splits

Dataset versions are immutable manifests built from approved examples. Splits are deterministic by source group so examples from the same trip/source do not cross train/eval boundaries. Golden benchmark cases marked `holdout` remain holdout and must not be used for training.

## Export

Exports require `AI_DATASET_EXPORT_ENABLED=true`. The package contains JSONL files for train/validation/test/holdout plus `manifest.json`, `checksums.txt`, and `README.md`. Exports are stored under `AI_DATASET_EXPORT_DIR` with private filesystem permissions and must be validated with `scripts/ai/validate-dataset-export.sh`.

## Audit and retention

Review actions are recorded in `ai_dataset_review_events`. Dataset examples and versions are retained for auditability until a future policy defines archival/deletion. Export directories are temporary operational artifacts and should be cleaned according to `AI_DATASET_EXPORT_RETENTION_DAYS`.

## Incident response

If private data is detected after approval or export:

1. Disable export by setting `AI_DATASET_EXPORT_ENABLED=false`.
2. Mark affected examples rejected or needs-changes.
3. Invalidate any affected dataset version.
4. Delete local export packages after preserving the audit record.
5. Record the issue, source, reviewer, and remediation in the incident log.
6. Add a sanitizer/scorer regression test before re-enabling export.
