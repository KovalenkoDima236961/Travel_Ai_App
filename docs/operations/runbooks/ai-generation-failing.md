# Runbook: AI generation failing

1. Determine the mode. `core` uses Trip mock generation; AI calls require the
   `ai` profile and the appropriate `TRIP_ITINERARY_GENERATOR_MODE`/AI settings.
2. Check `curl -fsS http://localhost:8000/ready`, AI and Worker logs, then
   Ollama `http://localhost:11434/api/tags` and container health. For RAG,
   inspect embedding/Chroma readiness and knowledge indexing.
3. Classify failure: timeout, unreachable model, invalid JSON/repair exhausted,
   schema validation, stale itinerary revision, cancellation, or provider/context
   dependency. Use job/trace IDs and correlation IDs, not raw prompts.
4. Restore mock mode or correct configuration/models, then retry only the
   affected safe job through the normal UI/ops route. A stale revision needs a
   fresh user decision; never overwrite a newer itinerary.
5. Prompts may contain personal travel data. Do not paste raw prompts, model
   responses, receipt text, tokens, or calendar data into tickets/logs. Keep
   `LOG_LLM_PAYLOADS` and prompt logging disabled unless an approved local-only
   diagnosis requires redacted telemetry.

See [AI generation guide](../../features/ai-generation.md).
