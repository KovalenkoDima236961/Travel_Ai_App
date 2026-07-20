# Trusted travel data providers

## Problem and decision

Curated knowledge in `data/travel-knowledge` is accurate but small. It covers four destinations, ages without anyone noticing, and often lacks opening hours and coordinates. Provider data fixes coverage and freshness, but it introduces duplicates, disagreement between sources, licensing obligations, and records that are simply wrong.

V1 therefore treats provider data as **evidence, not truth**. A provider result becomes an observation first, is normalized and scored, and only influences AI generation once it clears a quality threshold or a human approves it. Ingesting more data is not the goal; ingesting data whose trustworthiness is known is.

Ownership follows the existing split. Trip Service owns the normalized store, scoring, and review, because knowledge is planning input and validation evidence. Worker Service runs repeatable ingestion. Real network-backed adapters belong in External Integrations Service, behind its existing quota, cache, and rate-limit guards.

## Supported source categories

`travel_knowledge_sources.source_type` keeps the vocabulary defined in migration 000042:

| Category | Meaning |
| --- | --- |
| `manual_curated` | Original editorial records in `data/travel-knowledge` |
| `provider_place` | Results from a configured commercial provider adapter |
| `open_data` | Openly licensed sources such as OpenStreetMap or Wikidata |
| `user_approved_match` | A place a user explicitly confirmed during trip review |
| `user_feedback` | Aggregated signals; a modifier, never a standalone source |
| `mock_test_data` | Deterministic synthetic fixtures for local and CI use |

## Source trust model

Trust is a property of the source, not of an individual record, and it is the largest single term in the quality score.

| Trust level | Weight | Notes |
| --- | --- | --- |
| `trusted_curated` | 1.00 | Reviewed by the team |
| `trusted_provider` | 0.85 | Configured commercial provider with documented terms |
| `public_open_data` | 0.70 | Openly licensed; field completeness varies |
| `app_observed` | 0.55 | Derived from app flows |
| `user_feedback` | 0.45 | Adjusts confidence; never the sole basis for a record |
| `mock` | 0.40 | Capped below the strong-grounding threshold by design |
| `unknown` | 0.15 | Cannot reach strong grounding |

`mock` sits below `public_open_data` deliberately: synthetic fixtures must never outrank real data if they are ever ingested into a shared environment. `unknown` is the default for any provider that has not been registered with license and terms, so adding an adapter without documenting it degrades its data rather than trusting it silently.

## Licensing and attribution

Every non-curated record must carry a license name; adapters declare it through `LicenseInfo`, and `EnsureProviderSource` refuses to register a source without one. A record whose license is missing is capped at a quality score of 0.40, which places it below the weak-grounding floor — an unlicensed record is a policy failure, not merely a low-quality one.

Attribution is stored on the source and on the record. When a destination's grounding context is used, the attributions of contributing sources are returned alongside the places so any user-facing surface that needs to display them can. Provider terms must be reviewed before a real adapter is enabled, and the review outcome recorded in this document.

Prohibited outright: scraping arbitrary travel sites, copying guidebook or blog prose, and ingesting copyrighted description text without a license permitting it. Facts (a place exists, its coordinates, its category) are ingested; expressive text is not.

## Allowed and disallowed data

Ingestion stores names, aliases, categories, coordinates, addresses, websites, opening hours, ratings, price levels, tags, and provenance.

It must never store provider API keys or secrets, private user notes, comments, receipts or OCR text, calendar data, raw AI prompts, or any sensitive feedback metadata. These are not merely filtered at the API boundary — they never enter the observation path, because provider adapters build records from provider responses, not from application state.

## Raw payload policy

`travel_provider_place_observations.raw_payload` is **off by default**. It is retained only when the run policy opts in (`KNOWLEDGE_PROVIDER_STORE_RAW_PAYLOAD=true`) *and* the adapter's license permits it (`LicenseInfo.AllowsRawPayload`). The license restriction wins over the run policy, so a permissive environment setting cannot override provider terms.

Where retained, payloads are size-limited, must be free of secrets, and are exposed only to ops admins. Production deployments should leave raw payload storage disabled unless a specific debugging need justifies it.

## Normalization rules

Provider vocabulary is folded into the taxonomy this codebase already enforces through the `travel_places` category CHECK constraint and `allowedCategories`. `historical_site` and `religious_site` become `landmark`; `gallery` becomes `museum`; `nightlife` and `outdoor_activity` become `activity`; `shopping` becomes `market`; `accommodation` becomes `other`, since lodging is not a planning category here. The original label is preserved as `subcategory` so detail is not lost.

Names are normalized for matching only — accents folded, parentheticals and punctuation stripped, provider suffixes such as "Ticket Office" removed. The display name keeps its original form. Coordinates outside valid ranges, and the (0, 0) placeholder providers use as a default, are treated as missing rather than trusted. Website URLs keep only the http(s) scheme, host, and path; query strings and fragments are stripped because that is where tracking parameters and occasionally credential-like tokens live.

## Deduplication strategy

Matching is deterministic scoring, not a similarity model, because an incorrect automatic merge is expensive to reverse once a canonical record has been reindexed and referenced by generated itineraries.

Evidence is weighted: a provider reference match is conclusive (1.0); otherwise name and alias evidence carries 55%, coordinate proximity 25%, and category compatibility 20%. Two vetoes apply. A pair more than 5 km apart is capped below the match threshold regardless of name, so "Old Town" in two cities never merges. Weak name evidence is capped even when coordinates are close, so two neighbouring cafés stay distinct.

Thresholds: **≥0.90** auto-matches, **0.70–0.89** creates a duplicate group for review, **<0.70** is no match. When two candidates both clear the review threshold the observation goes to review rather than picking the higher score — ambiguity is precisely the case a human should resolve.

Merging is always an explicit Ops action. The canonical record keeps the highest-trust values for category and coordinates, but takes **opening hours from the freshest** record, since hours change far more often than a location does. Aliases, tags, and provider references are combined. Merged records keep their row for audit and for existing itinerary references, but are set to `review_status = 'merged'` with a zeroed quality score, so both retrieval paths exclude them.

## Quality scoring model

```text
qualityScore =
  0.25 sourceTrust + 0.20 destinationMatch + 0.15 coordinateCompleteness +
  0.10 categoryConfidence + 0.10 freshness + 0.10 providerAgreement +
  0.05 userFeedback + 0.05 completeness
  - duplicatePenalty - reviewPenalty
```

The result is then scaled by name quality, so a record named "Cafe" cannot reach strong grounding however complete its other fields are — an itinerary item a traveller cannot identify is not usable regardless of its metadata.

Provider agreement requires **independent** providers: a single observation scores a neutral 0.5, and repeated observations from one provider score 0.6, because one provider repeating itself is not corroboration. User feedback starts neutral at 0.5 and moves with sample size; a single unhappy user cannot flag a place, and two distinct users reporting problems are required before a record is sent to review.

Scoring is pure arithmetic over explicit inputs with no clock, randomness, or model call, so a score is reproducible and explainable in the Ops quality breakdown.

## Quality thresholds

| Setting | Default | Effect |
| --- | --- | --- |
| `KNOWLEDGE_AI_STRONG_MIN_QUALITY` | 0.75 | At or above: strong grounding |
| `KNOWLEDGE_AI_WEAK_MIN_QUALITY` | 0.55 | At or above: weak grounding, flagged for review |
| `KNOWLEDGE_NEEDS_REVIEW_BELOW_QUALITY` | 0.65 | Below: marked `needs_review` |
| `KNOWLEDGE_REJECT_BELOW_QUALITY` | 0.30 | Below: never persisted as a place |

Human decisions outrank automatic ones in both directions. An `approved` record is promoted to strong grounding, but approval cannot rescue a record below the weak floor. `rejected` and `merged` records are excluded from grounding regardless of score, and no job overwrites a human status.

## Freshness policy

Freshness decays linearly to zero at twice the category window, so a record does not fall off a cliff the day it goes stale:

| Category | Window |
| --- | --- |
| Transport | 30 days |
| Restaurants, cafés, markets | 45 days |
| Activities, unmapped categories | 90 days |
| Museums | 120 days |
| Landmarks, parks, viewpoints, neighborhoods, nature | 180 days |

Opening-hours-sensitive categories expire fastest because hours are the field most likely to be wrong.

## Refresh policy

`knowledge_provider_refresh_stale_places` selects active, non-rejected records whose `last_provider_refresh_at` is older than `KNOWLEDGE_REFRESH_STALE_AFTER_DAYS`. Batches are bounded by `KNOWLEDGE_REFRESH_BATCH_SIZE` and refresh is grouped by destination, so one destination costs one provider search rather than one request per stale place. There are no unbounded refresh jobs. Real adapters must additionally pass through the existing provider quota and rate-limit guard.

## Ops review workflow

Ops endpoints live under `/ops/ai/knowledge` and inherit the ops admin check and Ops Dashboard feature flag from the existing `/ops` route group. They expose ingestion status and runs, place listing and detail with filters (low quality, stale, missing coordinates, needs review, duplicates, rejected), review actions, per-place refresh, duplicate group listing with merge and reject, a quality summary, and provider observations.

Rejecting a record requires a reason: an unexplained rejection is not auditable. Every review and merge action writes a `travel_knowledge_review_events` row in the same transaction as the change, so a decision can never be recorded without its audit trail. Audit values are summaries only — no secrets, no raw payloads, no private user content.

## AI grounding usage rules

Retrieval excludes rejected and merged records **in SQL**, not in application code, so no caller can bypass the rule through a different code path. Strong records are ordered first and the result set is capped, because prompt space is finite and the best evidence should occupy it.

Each place carries its quality score, freshness, confidence, review status, grounding strength, and warnings. The prompt must prefer strong records; items built from weak records are marked `needsPlaceReview`; rejected and merged records are never present to be used. When a destination's coverage is low, generation reports `partial` quality and the user-facing review says "Limited verified place data for this destination." rather than inventing plausible place names to fill the gap.

## Destination coverage

Coverage combines high-quality volume (35%), the high-quality share of records (25%), core category coverage (20%), freshness (10%), and coordinate completeness (10%). High-quality count dominates deliberately: ten stale, low-quality records do not make a destination well covered. Scores at or above 0.70 are `available`, 0.35–0.69 `partial`, and below that `limited`.

## Local and CI behaviour

`KNOWLEDGE_PROVIDER` defaults to `mock`. `MockKnowledgeProvider` serves fixed fixtures for Rome, Paris, Vienna, and Bratislava, anchored to `MockReferenceTime` so observation timestamps — and therefore freshness scores — are identical on every run.

The fixtures deliberately include a Colosseum duplicate under its Italian name, a record with no coordinates, a record stale past its TTL, cross-provider disagreement about the Eiffel Tower's category, and a uselessly generic "Cafe". These are the cases the deduplication, review, and refresh tests exercise; a test asserts they still exist, so the edge cases cannot silently disappear.

No real provider is called in CI. A configured-but-unavailable real provider either falls back to mock or fails loudly — it never attempts a network call from the test suite.

## Configuration

```bash
KNOWLEDGE_PROVIDER=mock|foursquare|opentripmap|wikidata
KNOWLEDGE_PROVIDER_FALLBACK_TO_MOCK=true
KNOWLEDGE_PROVIDER_TIMEOUT_SECONDS=8
KNOWLEDGE_PROVIDER_MAX_RESULTS_PER_DESTINATION=100
KNOWLEDGE_PROVIDER_REFRESH_ENABLED=true
KNOWLEDGE_PROVIDER_STORE_RAW_PAYLOAD=false
KNOWLEDGE_REFRESH_STALE_AFTER_DAYS=30
KNOWLEDGE_REFRESH_BATCH_SIZE=100
KNOWLEDGE_AI_STRONG_MIN_QUALITY=0.75
KNOWLEDGE_AI_WEAK_MIN_QUALITY=0.55
KNOWLEDGE_NEEDS_REVIEW_BELOW_QUALITY=0.65
KNOWLEDGE_REJECT_BELOW_QUALITY=0.30
```

A threshold outside 0..1 or an unparseable value is ignored in favour of the default, so a configuration mistake cannot silently disable the quality gate.

## Adding a real provider

A provider may not be enabled until its license, attribution requirement, terms URL, rate limits, and raw-payload stance are documented here and encoded in its `LicenseInfo`. Adapters that make network calls belong in External Integrations Service, using the existing provider config, quota guard, cache, and timeout patterns, and must be disabled in CI. See `docs/development/playbooks/add-travel-data-provider.md`.

No real provider adapter is enabled in v1. Mock ingestion plus the observation pipeline is the shipped scope.

## Vector indexing and merges

There is no reindex step after a merge, and that is deliberate rather than missing.

Provider-backed places reach the model through `RetrieveGrounding`, which reads `travel_places` in SQL and excludes rejected and merged records in the query itself. Provider places are never written to the Chroma collection — that collection holds curated *documents*, whose ingestion path is unchanged. A merge therefore takes effect the moment the rows change, and there is no embedding to invalidate.

`knowledge_reindex_after_merge` exists as an addressable job name and currently re-runs the ingestion pass (which rematches observations and recomputes scores). If provider places are ever added to the vector index, that job is where the reindex call belongs, and the `ReindexRequired` flag on `IngestResult` is the signal it should consume.

## What v1 does and does not include

Shipped and verified:

- Schema, normalization, scoring, matching, deduplication, merge, review, and audit, exercised against a live PostgreSQL instance (tested on 14; the SQL idioms used are also valid on the 16 image used in deployment).
- Mock provider ingestion, idempotent and deterministic, with Worker CLI jobs.
- Ops endpoints under `/ops/ai/knowledge`, wired into the composition root.
- Quality-filtered grounding retrieval, attached to the generation request and consumed by the AI Planning prompt.

Deferred to a follow-up:

- **Ops UI panel.** The endpoints exist and are tested; the Ops Dashboard screen that consumes them is not built.
- **Destination coverage in generation output metadata.** Coverage is computed, returned by retrieval, and sent into the prompt (including the limited-coverage instruction), but it is not yet threaded into the `generationQuality` metadata on the generation *result* that the post-generation review screen reads.
- **Prometheus metrics.** The counters listed for provider requests, ingestion outcomes, and quality distribution are specified but not emitted; structured logs cover the same events today.
- **Trip Service OpenAPI entries for the new ops routes** and the corresponding endpoint-inventory rows. The AI Planning contract snapshot and generated web types *are* current.
- **Frontend and Playwright coverage** for the review and merge flows.

No real provider adapter is enabled, by design: mock plus the observation pipeline is the shipped v1 scope.
