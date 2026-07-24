# Build an AI dataset version

1. Run `scripts/ai/validate-training-examples.sh` for local manual examples.
2. Import golden holdout cases with `POST /ops/ai/datasets/projects/{projectId}/extract-golden`.
3. Import manual examples with `POST /ops/ai/datasets/projects/{projectId}/extract-manual`.
4. Review and approve eligible examples.
5. Check readiness:

```bash
ALLOW_NOT_READY=1 scripts/ai/run-fine-tuning-readiness-check.sh
```

6. Build a semantic version with `POST /ops/ai/datasets/projects/{projectId}/versions`.
7. Inspect the manifest split counts and checksum.
8. Export only when explicitly enabled with `AI_DATASET_EXPORT_ENABLED=true`.
9. Validate the export:

```bash
scripts/ai/validate-dataset-export.sh data/ai-datasets/<export-dir>
```

Do not use the holdout split for training.
