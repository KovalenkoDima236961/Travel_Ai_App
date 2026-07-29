# OpenAI Alpha Integration

OpenAI is integrated through AI Planning Service only. Web, Trip Service, and
Worker Service stay provider-neutral and must never receive an OpenAI SDK object
or `OPENAI_API_KEY`.

## Architecture

```
Trip Service / Worker Service
  -> AI Planning Service
    -> AIModelProvider
      -> OpenAIProvider
      -> OllamaProvider
      -> MockProvider
```

The provider interface lives in `services/ai-planning-service/app/providers`.
The synchronous service adapters keep existing API routes stable while the
OpenAI provider uses the async SDK internally.

## Official API Basis

The implementation follows the current OpenAI Responses API, Structured Outputs
guide, Python SDK, data-control guidance, and Batch API guidance:

- https://platform.openai.com/docs/api-reference/responses/create
- https://platform.openai.com/docs/guides/structured-outputs
- https://github.com/openai/openai-python
- https://platform.openai.com/docs/models/default-usage-policies-by-endpoint
- https://platform.openai.com/docs/guides/batch

Model names are configuration values because supported models and capabilities
change over time.

## Configuration

Local and CI default to deterministic mock mode:

```bash
AI_MODEL_PROVIDER=mock
AI_MODEL_PROVIDER_FALLBACK=none
OPENAI_ENABLED=false
```

Alpha staging or production selects OpenAI explicitly:

```bash
AI_MODEL_PROVIDER=openai
ITINERARY_GENERATOR_MODE=openai
COPILOT_MODE=openai
TRIP_RECAP_AI_MODE=openai
AI_MODEL_PROVIDER_FALLBACK=mock
OPENAI_ENABLED=true
OPENAI_API_KEY=...
OPENAI_STORE_RESPONSES=false
OPENAI_MODEL_DEFAULT=...
```

Operation-specific model aliases override `OPENAI_MODEL_DEFAULT`:

```bash
OPENAI_MODEL_ITINERARY=
OPENAI_MODEL_REGENERATION=
OPENAI_MODEL_REPAIR=
OPENAI_MODEL_DISCOVERY=
OPENAI_MODEL_ROUTE_ALTERNATIVES=
OPENAI_MODEL_BUDGET_OPTIMIZATION=
OPENAI_MODEL_CHECKLIST=
OPENAI_MODEL_COPILOT=
OPENAI_MODEL_RECAP=
OPENAI_MODEL_EVALUATION=
```

## Runtime Guarantees

Implemented now:

- provider-neutral result metadata with request ID, model, token usage, latency,
  fallback metadata, and safe provider metadata
- OpenAI Responses API structured output path backed by Pydantic models
- one reusable async OpenAI client per app instance
- startup validation for provider, fallback, OpenAI enabled state, API key, model
  aliases, timeouts, retry bounds, and explicit production storage behavior
- privacy sanitizer before model-bound content leaves AI Planning Service
- bounded repair for itinerary generation paths that already support repair
- safe response metadata on private AI Planning responses
- Prometheus metrics for provider requests, latency, tokens, retries, errors, and
  fallbacks

Still external to this provider layer:

- Trip Service persistence, revisions, activity events, notifications, and public
  response filtering
- durable usage ledger and estimated cost accounting
- distributed spend limits, rate limits, and circuit breaker state
- Ops Dashboard panels and mutable feature flags

Do not claim those controls are complete until their owning services implement
and test them.

