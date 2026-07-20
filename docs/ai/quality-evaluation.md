# AI itinerary quality evaluation

## Golden cases

The versioned cases in `evals/ai-itinerary/cases` cover Rome food/culture, rainy Paris, low-walking Vienna, budget Bratislava, Slovakia nature, Barcelona family, a Vienna–Bratislava–Budapest train route, and a senior-friendly trip. They use curated data and mock generation in CI; live/Ollama runs are optional local diagnostics.

## Metrics and scoring

The runner records `groundedPlaceRate`, `hallucinatedPlaceCount`, `destinationMismatchCount`, `duplicatePlaceCount`, `missingCoordinateCount`, `unrealisticDurationCount`, `overpackedDayCount`, `openingHoursRiskCount`, `budgetPlausibilityScore`, `routePlausibilityScore`, `varietyScore`, `preferenceMatchScore`, `schemaValidity`, `repairNeeded`, and `overallScore`.

`overallScore` is a transparent bounded score rather than a guarantee: schema validity and grounded coverage earn points; hallucination, mismatch, duplicates, unrealistic durations, and overpacked days deduct points. A quality change should compare the same cases, curated knowledge revision, and mock mode. No score replaces manual review of representative plans.

## Running and review

Run `./scripts/ai/validate-knowledge.sh` first, then `./scripts/ai/run-itinerary-evals.sh`. Reports are written to `evals/ai-itinerary/reports/<timestamp>.json` and `latest.md`. CI runs the same mock-only mode and never makes provider or Ollama calls.

Review regressions where overall score falls, any hallucinated or destination-mismatched place appears, or schema validity is false. Confirm changes against source provenance, licensing, realistic durations, weather handling, budget language, and user-facing wording. Keep known limitations visible: opening hours and availability can be stale, provider data can conflict, and a grounded itinerary still requires traveler verification.

## Provider data quality gates

Evaluation now runs against a knowledge base that includes provider-backed records, so an evaluation regression can come from data as easily as from a prompt change. When investigating a regression, check the knowledge quality summary (`GET /ops/ai/knowledge/quality-summary`) alongside the prompt diff: a drop in grounded-place rate often means records went stale or were rejected, not that generation got worse.

Grounding strength is the link between data quality and evaluation. Only records at or above `KNOWLEDGE_AI_STRONG_MIN_QUALITY` are offered as strong grounding; weak records produce items marked `needsPlaceReview`, and rejected or merged records are excluded entirely. An evaluation run whose destinations have low coverage should be read as a coverage result, not a model result.

Mock-mode evaluation stays deterministic. `MockKnowledgeProvider` anchors every observation to `MockReferenceTime`, so freshness scores — and therefore quality scores and grounding strength — are identical on every run. See [trusted travel data providers](trusted-travel-data-providers.md).
