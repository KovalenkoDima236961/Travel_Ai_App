# AI itinerary golden evaluations

Each case is a stable product scenario, not a source of training labels. The mock-mode runner loads curated destination records, requests an itinerary with an explicit compact grounding context, and scores the response without network access.

Run from the repository root:

```bash
./scripts/ai/validate-knowledge.sh
./scripts/ai/run-itinerary-evals.sh
```

Set `AI_EVAL_MODE=ollama` only for an optional local comparison after starting the AI stack. CI always runs the mock mode. Generated reports are ignored by Git except `latest.md`, which is an example of the current deterministic baseline.
