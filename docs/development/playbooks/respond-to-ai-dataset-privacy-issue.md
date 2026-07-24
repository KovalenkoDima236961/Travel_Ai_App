# Respond to an AI dataset privacy issue

Use this when private data, raw prompts, hidden system instructions, raw provider text without license, or revoked-consent examples are found in a candidate, version, or export.

1. Disable export:

```bash
AI_DATASET_EXPORT_ENABLED=false
```

2. Reject or mark affected examples as needs-changes.
3. Invalidate any version that included the affected examples.
4. Remove local export directories after recording their path, version ID, checksum, and reviewer.
5. Preserve `ai_dataset_review_events` and relevant service logs for audit.
6. Add a regression test to `services/trip-service/internal/aidataset` for the missed private-data pattern.
7. Re-run:

```bash
go test ./internal/aidataset
scripts/ai/validate-training-examples.sh
```

8. Rebuild/export only after governance review approves the fix.
