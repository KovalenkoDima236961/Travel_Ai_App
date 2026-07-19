# Fine-tuning readiness

V1 does not fine-tune a model. A future dataset may contain only approved, sanitized examples with a documented license, source provenance, purpose, quality review, and explicit training consent where user-derived. It must not include raw private itineraries, comments, receipts/OCR, calendars, emails, notes, passwords, tokens, API keys, or raw prompts.

Before a LoRA or other adaptation, require a representative and deduplicated dataset, evaluator coverage for regressions, a documented minimum benchmark threshold, privacy/security review, model-card update, human spot checks, and a production rollback route to the current base model. Hold out evaluation cases; never train on them. If a candidate worsens hallucination, destination matching, safety, or preference fit, disable it and retain the benchmark/report for investigation.
