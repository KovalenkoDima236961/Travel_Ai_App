# Playbook: review knowledge duplicates

Duplicate groups are proposals from `knowledge_duplicate_detection`, never automatic merges. A wrong merge is expensive to reverse once the canonical record has been reindexed and referenced by generated itineraries, so the decision is yours.

## Triage

1. Open the Ops AI Knowledge Quality panel and filter by **duplicates**, or call `GET /ops/ai/knowledge/duplicates?destinationId=<id>`.
2. Each group carries the confidence and the reasons that produced it. Confidence 0.70–0.89 means the matcher was uncertain; 0.90+ usually means an exact name or alias match with close coordinates.
3. Open each member's detail drawer and compare canonical name, aliases, category, coordinates, opening hours, source, and provider references.

## Deciding

4. **Merge** when the members describe the same physical place — for example the same landmark under a local-language name.
5. **Reject** when they are genuinely distinct: two cafés on one street, or a museum and the palace containing it. Rejecting unlinks the members and records that they are distinct, so detection does not re-raise the same finding.
6. When unsure, leave the group open and add a reason. An open group does not block grounding; both records remain usable according to their own quality scores.

## Merging

7. Choose the canonical record. Prefer the highest-trust source, then the highest quality score. The merge resolver keeps the canonical record's category and coordinates but takes **opening hours from the freshest** member, since hours change more often than location.
8. Call `POST /ops/ai/knowledge/duplicates/{groupId}/merge` with `canonicalPlaceId` and a reason, or use the panel's merge action.
9. Verify the outcome: aliases, tags, and provider references are combined onto the canonical record; absorbed records become `review_status = 'merged'` with a zeroed quality score, are archived, and are excluded from grounding retrieval. Their observations are repointed at the canonical record.
10. Confirm the audit event exists in `travel_knowledge_review_events`. The merge and its audit row are written in one transaction, so a missing event means the merge did not happen.

## After merging

11. The canonical record's text changed, so it needs reindexing. Run the reindex job (`knowledge_reindex_after_merge`) or `./scripts/ai/reindex-knowledge.sh` for the affected destination.
12. Spot-check retrieval for the destination: `GET /ops/ai/knowledge/places?destinationId=<id>` should show one canonical record and no merged record in grounding-eligible state.

## If you merged the wrong records

There is no automatic unmerge. Recover by setting the absorbed record's `review_status` back to `needs_review` via the review endpoint, correcting the canonical record's fields, and re-running quality scoring. Record what happened in the reason field — the audit trail is how the next reviewer understands the state.
