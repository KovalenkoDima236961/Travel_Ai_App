# Playbook: add destination coverage

Use this when a destination generates weak itineraries because it has too few verified places. Coverage below 0.70 makes generation report `partial` quality and shows users "Limited verified place data for this destination."

## Diagnose first

1. Check the destination's coverage: `GET /ops/ai/knowledge/quality-summary`, or read `coverage` in the grounding result. The warnings name the specific gap — missing categories, stale records, or missing coordinates.
2. Identify which term is low. Coverage weights high-quality volume at 35%, high-quality share at 25%, core category coverage at 20%, freshness at 10%, and coordinates at 10%. Ten stale low-quality records do not make a destination covered, so adding volume alone often will not help.

## Fill the gap

3. **Missing categories** (the usual cause): the core set is landmark, museum, park, restaurant, neighborhood. Add curated records for the missing categories following [add curated destination knowledge](add-curated-destination-knowledge.md). Curated data has the highest trust weight, so it moves coverage fastest.
4. **Stale records**: run the refresh job rather than re-adding data.
   ```bash
   go run ./cmd/knowledge-provider --job knowledge_provider_refresh_stale_places \
     --destination-id <uuid>
   ```
5. **Missing coordinates**: filter the Ops panel by **missing coordinates**. Coordinates are 15% of each record's quality score, so filling them raises both record quality and coverage.
6. **Thin overall**: run provider ingestion for the destination, then review what it produced. Do not approve records in bulk — the review step is what keeps quality meaningful.
   ```bash
   go run ./cmd/knowledge-provider --job knowledge_provider_ingest_destination \
     --destination "<name>" --country-code <XX> --dry-run
   ```
   Inspect the dry-run scores, then re-run without `--dry-run`.

## Review and verify

7. Work the **needs review** filter in the Ops panel. Approving a record promotes it to strong grounding, but approval cannot rescue a record below the weak floor — fix the underlying data instead.
8. Reject records that are wrong, with a reason. A rejection without a reason is refused, because it is not auditable.
9. Re-check coverage. Aim for `available` (0.70+), which needs roughly 12 high-quality places spread across the core categories.
10. Reindex the destination so the new records reach retrieval: `./scripts/ai/reindex-knowledge.sh`.
11. Generate a test itinerary for the destination and confirm items reference real records and that limited-coverage messaging no longer appears.

## When coverage cannot be raised

Some destinations genuinely lack good open data. That is an acceptable outcome: the system degrades to generic activities with an honest coverage warning. Do not lower the quality thresholds to make a destination look covered — that trades a visible limitation for an invisible correctness problem.
