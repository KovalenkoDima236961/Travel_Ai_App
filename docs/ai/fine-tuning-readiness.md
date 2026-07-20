# Fine-tuning readiness

V1 does not fine-tune a model. A future dataset may contain only approved, sanitized examples with a documented license, source provenance, purpose, quality review, and explicit training consent where user-derived. It must not include raw private itineraries, comments, receipts/OCR, calendars, emails, notes, passwords, tokens, API keys, or raw prompts.

Before a LoRA or other adaptation, require a representative and deduplicated dataset, evaluator coverage for regressions, a documented minimum benchmark threshold, privacy/security review, model-card update, human spot checks, and a production rollback route to the current base model. Hold out evaluation cases; never train on them. If a candidate worsens hallucination, destination matching, safety, or preference fit, disable it and retain the benchmark/report for investigation.

## Provider data and training eligibility

Provider-derived knowledge records are subject to the same licensing rules as any other candidate training data, and they are stricter than they look: a record's license permits *this application* to use the facts for planning, which is not the same as permitting model training on the provider's text.

For any future dataset work, reference provider facts through grounding IDs rather than copying provider text, and exclude records whose source license is unknown or incompatible. Records with `review_status` of `rejected` or `merged`, and records below the weak-grounding quality floor, are not eligible candidates — they are already excluded from influencing generation and should not re-enter through a dataset.

Mock provider fixtures are synthetic and clearly labelled. They are usable for pipeline testing and must never be exported as if they were real travel data. See [trusted travel data providers](trusted-travel-data-providers.md).
