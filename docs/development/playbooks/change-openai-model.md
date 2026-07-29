# Change OpenAI Model

Model names are configuration, not domain logic.

## Steps

1. Pick the operation alias to change, for example `OPENAI_MODEL_ITINERARY`.
2. Verify the model supports the required structured-output behavior.
3. Update staging environment configuration only.
4. Run:

```bash
./scripts/ai/check-openai-config.sh --env-file infra/.env.staging.example
./scripts/ai/run-alpha-provider-evals.sh --mock
```

5. Run a protected real OpenAI evaluation in staging with explicit approval and
   request limits.
6. Compare against the previous report and Ollama baseline.
7. Promote only if schema validity, grounding, fallback, latency, and cost gates
   pass.

Do not hardcode the model name in Python, Go, TypeScript, Docker images, or API
contracts.

