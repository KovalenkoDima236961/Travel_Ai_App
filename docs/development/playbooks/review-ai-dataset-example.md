# Review an AI dataset example

1. Open the Ops dashboard and inspect the AI dataset curation panel.
2. Confirm `consentStatus` is `granted` or `not_required`.
3. Confirm `sanitizationStatus` is `passed`.
4. Confirm `qualityStatus` is `passed` and the score meets the project threshold.
5. Inspect input, grounding, expected output, labels, and provenance through the ops API if the summary is insufficient.
6. Reject examples that include private data, hidden prompts, raw provider text without an allowed license, hallucinated places, impossible schedules, or unclear provenance.
7. Approve only with a review reason that explains why the example is training-eligible.

Relevant endpoints:

- `GET /ops/ai/datasets/examples`
- `POST /ops/ai/datasets/examples/{exampleId}/approve`
- `POST /ops/ai/datasets/examples/{exampleId}/reject`
- `POST /ops/ai/datasets/examples/{exampleId}/resanitize`
- `POST /ops/ai/datasets/examples/{exampleId}/rescore`
