# OllamaItineraryGenerator

> God node · 44 connections · `services/ai-planning-service/app/services/ollama_itinerary_generator.py`

## Connections by Relation

### contains
- [[ollama_itinerary_generator.py]] `EXTRACTED`

### method
- [[._generate_with_ollama()]] `EXTRACTED`
- [[._regenerate_day_with_ollama()]] `EXTRACTED`
- [[._regenerate_item_with_ollama()]] `EXTRACTED`
- [[._record_generation_error()]] `EXTRACTED`
- [[._call_ollama()]] `EXTRACTED`
- [[._repair_is_enabled()]] `EXTRACTED`
- [[.generate()]] `EXTRACTED`
- [[.regenerate_day()]] `EXTRACTED`
- [[.regenerate_item()]] `EXTRACTED`
- [[._base_partial_log_context()]] `EXTRACTED`
- [[._get_destination_context_for()]] `EXTRACTED`
- [[._get_partial_rag_chunks()]] `EXTRACTED`
- [[._validation_error_for_prompt()]] `EXTRACTED`
- [[._log_llm_payload()]] `EXTRACTED`
- [[._duration_ms()]] `EXTRACTED`
- [[._base_log_context()]] `EXTRACTED`
- [[._get_destination_context()]] `EXTRACTED`
- [[._get_rag_chunks()]] `EXTRACTED`
- [[._parse_and_validate()]] `EXTRACTED`
- [[._post_to_ollama()]] `EXTRACTED`

### uses
- [[MockItineraryGenerator]] `INFERRED`
- [[LLMResponseParseError]] `INFERRED`
- [[Settings]] `INFERRED`
- [[ItineraryGenerator]] `INFERRED`
- [[KnowledgeSearchService]] `INFERRED`
- [[GenerateItineraryRequest]] `INFERRED`
- [[ItineraryResponse]] `INFERRED`
- [[StaticDestinationKnowledgeProvider]] `INFERRED`
- [[StaticKnowledgeSearchService]] `INFERRED`
- [[KnowledgeSearchResult]] `INFERRED`
- [[FailingDestinationKnowledgeProvider]] `INFERRED`
- [[DestinationContext]] `INFERRED`
- [[ItineraryGenerationError]] `INFERRED`
- [[RegenerateDayRequest]] `INFERRED`
- [[RegenerateDayResponse]] `INFERRED`
- [[RegenerateItemResponse]] `INFERRED`
- [[DestinationKnowledgeProvider]] `INFERRED`
- [[ItineraryValidationError]] `INFERRED`
- [[RegenerateItemRequest]] `INFERRED`
- [[ItineraryValidator]] `INFERRED`

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*