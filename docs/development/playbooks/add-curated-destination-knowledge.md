# Add curated destination knowledge

1. Add a source with license/attribution to `data/travel-knowledge/sources.json`; use `manual_curated` only for original concise editorial content.
2. Add `destinations/<city>.json` using the checked-in schema. Every place needs a category, confidence, and source key. Do not copy guidebook/blog prose or include private user content.
3. Add a short original `documents/<city>.en.md` with planning constraints rather than promotional text.
4. Run `./scripts/ai/validate-knowledge.sh` and `./scripts/ai/ingest-knowledge.sh --dry-run`.
5. Add or update a golden evaluation case if the destination changes user-visible planning coverage. Review the mock report before enabling the data for production grounding.

Provider/open-data work additionally requires terms, attribution, retention, and rate-limit review. A record with ambiguous ownership, missing license, or insufficient confidence must stay out of strong grounding.
