# OpenAI Alpha Evaluations

This directory tracks alpha comparison work for mock, Ollama, and OpenAI
providers.

- `cases/` holds sanitized or synthetic evaluation cases only.
- `reports/` holds generated reports. Do not store raw prompts or raw provider
  responses.

Default CI evaluation remains deterministic mock mode through:

```bash
./scripts/ai/run-alpha-provider-evals.sh --mock
```

Real OpenAI runs require explicit `--allow-real-openai`, `OPENAI_ENABLED=true`,
`OPENAI_API_KEY`, model configuration, request-count limits, and spend review.

