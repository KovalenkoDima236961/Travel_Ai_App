# Run OpenAI Alpha Evaluation

Mock evaluation is always safe and CI-compatible:

```bash
./scripts/ai/run-alpha-provider-evals.sh --mock
```

Ollama comparison requires the local AI stack and model:

```bash
AI_EVAL_MODE=ollama ./scripts/ai/run-alpha-provider-evals.sh --ollama
```

Real OpenAI evaluation must be run only in local or staging with explicit
approval, request limits, and spend caps:

```bash
OPENAI_ENABLED=true \
OPENAI_API_KEY=... \
OPENAI_MODEL_DEFAULT=... \
OPENAI_STORE_RESPONSES=false \
./scripts/ai/run-alpha-provider-evals.sh --openai --allow-real-openai --max-requests 5
```

Reports belong under `evals/alpha-openai/reports/`. Do not commit raw prompts,
raw provider responses, API keys, or private trip data.

