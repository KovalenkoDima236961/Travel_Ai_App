# Ollama Generator Tests

> 29 nodes · cohesion 0.18

## Key Concepts

- **test_ollama_generator.py** (31 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **_settings()** (22 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **_request()** (18 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **_itinerary_body()** (9 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_injects_destination_context_into_initial_and_repair_prompts()** (5 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_log_llm_payloads_false_does_not_log_full_prompt_or_response()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_log_llm_payloads_true_in_development_allows_payload_logging()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_log_llm_payloads_true_outside_development_does_not_log_payloads()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_calls_rag_search_and_injects_chunks_when_enabled()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_does_not_lookup_destination_context_when_disabled()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_parses_markdown_fenced_json_response_successfully()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_wrong_number_of_days_triggers_repair_and_returns_repaired_itinerary()** (4 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_generated_response_keeps_exact_itinerary_response_shape()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_initial_invalid_json_and_repair_disabled_falls_back_when_fallback_enabled()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_initial_invalid_json_and_repair_enabled_triggers_repair()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_falls_back_to_mock_when_request_fails_and_fallback_enabled()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_parses_valid_json_response_successfully()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_sends_request_to_api_generate()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_repair_response_invalid_and_fallback_disabled_raises_generation_error()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_repair_response_invalid_and_fallback_enabled_returns_mock_itinerary()** (3 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_missing_ollama_base_url_in_ollama_mode_fails_clearly()** (2 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_missing_ollama_model_in_ollama_mode_fails_clearly()** (2 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_mock_mode_factory_still_uses_mock_generator()** (2 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_factory_ignores_missing_destination_context_dir_when_enabled()** (2 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **test_ollama_mode_returns_error_when_request_fails_and_fallback_disabled()** (2 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- *... and 4 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/ai-planning-service/tests/test_ollama_generator.py`

## Audit Trail

- EXTRACTED: 154 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*