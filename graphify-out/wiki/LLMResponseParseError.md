# LLMResponseParseError

> God node · 23 connections · `services/ai-planning-service/app/services/llm_response_parser.py`

## Connections by Relation

### calls
- [[_parse_json()]] `EXTRACTED`
- [[parse_regenerate_day_response()]] `EXTRACTED`
- [[parse_regenerate_item_response()]] `EXTRACTED`
- [[parse_itinerary_response()]] `EXTRACTED`
- [[_ensure_exact_day_response_shape()]] `EXTRACTED`
- [[_ensure_exact_item_response_shape()]] `EXTRACTED`
- [[_ensure_exact_item_shape()]] `EXTRACTED`
- [[_ensure_item_values_valid()]] `EXTRACTED`
- [[_extract_first_json_object()]] `EXTRACTED`
- [[_ensure_exact_response_shape()]] `EXTRACTED`

### contains
- [[llm_response_parser.py]] `EXTRACTED`

### inherits
- [[ValueError]] `EXTRACTED`

### rationale_for
- [[Raised when an LLM response cannot be parsed into an itinerary.]] `EXTRACTED`

### uses
- [[OllamaItineraryGenerator]] `INFERRED`
- [[OllamaClientError]] `INFERRED`
- [[Raised when the Ollama API cannot provide a usable response.]] `INFERRED`
- [[ItineraryResponse]] `INFERRED`
- [[StaticDestinationKnowledgeProvider]] `INFERRED`
- [[StaticKnowledgeSearchService]] `INFERRED`
- [[FailingDestinationKnowledgeProvider]] `INFERRED`
- [[RegenerateDayResponse]] `INFERRED`
- [[RegenerateItemResponse]] `INFERRED`
- [[ItineraryItem]] `INFERRED`

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*