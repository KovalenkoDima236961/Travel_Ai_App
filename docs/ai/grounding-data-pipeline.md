# AI grounding data pipeline

## Problem and decision

An itinerary model can produce plausible text without proving that a named place exists, belongs in the requested destination, or is practical to visit. Fine-tuning before reliable labels, provenance, validation, and evaluation would amplify those defects. V1 therefore grounds generation in a small, auditable knowledge store and keeps generation fail-open when that store is temporarily unavailable.

Trip Service owns normalized knowledge records because they are planning inputs and validation evidence. Worker Service performs repeatable ingestion and indexing work. AI Planning Service retrieves compact, sanitized context and never treats retrieved text as instructions.

## Permitted data

V1 accepts original manually curated JSON/Markdown in `data/travel-knowledge`, synthetic test fixtures, user-approved matches, approved provider place records, and appropriately attributed open data. Provider and open-data adapters must record source URL, license/terms, attribution when required, confidence, and verification time.

It rejects arbitrary web scraping, copied guidebook or blog text, private comments, emails, calendar details, receipts/OCR, private notes, passwords/tokens, provider keys, and raw prompts. User feedback is stored as a restricted signal, not source content and not training data by default.

## Data model and provenance

`travel_knowledge_sources` identifies the source, license, attribution, trust level, and enabled state. Destinations, places, documents, chunks, and feedback signals have their own normalized records. Checksums make source imports idempotent. All non-curated records require a source and attribution; secret-like values and private content are rejected before persistence.

Place confidence is a value from 0 to 1. `active`, `archived`, and `rejected` are the only knowledge-record states. Generation only receives active places at or above the configured confidence threshold. Provenance is retained in private metadata and summarized to users; raw documents are never exposed on public shares.

## Ingestion and indexing flow

```text
curated JSON/Markdown or approved provider result
  -> validate + normalize + checksum + deduplicate
  -> Trip Service knowledge tables
  -> chunk documents / compact place descriptions
  -> Chroma collection with destination, source, license, tags, checksum metadata
  -> hybrid retrieval (destination filter + vector search, SQL fallback)
```

Worker job names are `knowledge_ingest_curated`, `knowledge_ingest_provider_places`, `knowledge_index_embeddings`, `knowledge_reindex_destination`, and `knowledge_quality_check`. Repeating an input only updates its checksum-derived rows. Dedupe starts with destination plus normalized name or provider reference, then uses alias and coordinate-proximity evidence. Ambiguous matches are preserved for review rather than silently merged.

## Prompt, validation, and feedback flow

Trip Service passes compact destination facts, high-confidence places, duration and weather hints, and retrieval warnings as `groundingContext`. AI prompts prefer those places, must not invent a specific place name, and must mark a generic activity `needsPlaceReview`. Generation output carries source, place ID, confidence, and warnings per item.

After generation, Trip Service validates destination, match confidence, coordinates, duplicate use, duration, and known opening-hours risk. It may make one targeted repair attempt when the configured bad-place threshold is exceeded. This validation is advisory-safe: it does not claim provider availability or correctness.

Explicit and implicit item feedback is sanitized to safe identifiers and bounded metadata. The default `consent_for_training` is false. Aggregates can improve evaluation, but future training requires explicit consent, policy review, and a separate approval process.

## Evaluation and future readiness

Golden cases in `evals/ai-itinerary` run deterministically in mock mode with no network, Ollama, or provider requirement. They report grounded-place rate, hallucinated/mismatched/duplicate counts, schedule risks, preference fit, and an overall score. Compare reports before/after a change; investigate a regression rather than accepting a higher text-quality impression.

Fine-tuning is not part of V1. It becomes eligible only when records have documented rights and consent, provenance, redaction, a representative benchmark, passing evaluation thresholds, a rollback plan, and manual review samples.
