# Itinerary Domain Models

> 153 nodes · cohesion 0.03

## Key Concepts

- **OllamaItineraryGenerator** (44 connections) — `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- **itinerary.py** (32 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **MockItineraryGenerator** (31 connections) — `services/ai-planning-service/app/services/itinerary_generator.py`
- **OllamaClientError** (21 connections) — `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- **Settings** (19 connections) — `services/ai-planning-service/app/config.py`
- **Raised when the Ollama API cannot provide a usable response.** (18 connections) — `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- **ItineraryGenerator** (16 connections) — `services/ai-planning-service/app/services/itinerary_generator.py`
- **KnowledgeSearchService** (15 connections) — `services/ai-planning-service/app/services/knowledge_search.py`
- **GenerateItineraryRequest** (14 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **APIModel** (13 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **ItineraryResponse** (13 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **StaticDestinationKnowledgeProvider** (12 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **StaticKnowledgeSearchService** (12 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **BaseModel** (11 connections)
- **KnowledgeSearchResult** (11 connections) — `services/ai-planning-service/app/schemas/knowledge.py`
- **FailingDestinationKnowledgeProvider** (11 connections) — `services/ai-planning-service/tests/test_ollama_generator.py`
- **DestinationContext** (10 connections) — `services/ai-planning-service/app/schemas/destination_context.py`
- **._generate_with_ollama()** (10 connections) — `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- **DestinationKnowledgeProvider** (9 connections) — `services/ai-planning-service/app/services/destination_knowledge.py`
- **ItineraryGenerationError** (9 connections) — `services/ai-planning-service/app/core/errors.py`
- **RegenerateDayRequest** (9 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **RegenerateDayResponse** (9 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **RegenerateItemResponse** (9 connections) — `services/ai-planning-service/app/schemas/itinerary.py`
- **ItineraryValidationError** (9 connections) — `services/ai-planning-service/app/services/itinerary_validator.py`
- **._regenerate_day_with_ollama()** (9 connections) — `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- *... and 128 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/ai-planning-service/app/config.py`
- `services/ai-planning-service/app/core/errors.py`
- `services/ai-planning-service/app/schemas/destination_context.py`
- `services/ai-planning-service/app/schemas/itinerary.py`
- `services/ai-planning-service/app/schemas/knowledge.py`
- `services/ai-planning-service/app/services/destination_knowledge.py`
- `services/ai-planning-service/app/services/itinerary_generator.py`
- `services/ai-planning-service/app/services/itinerary_validator.py`
- `services/ai-planning-service/app/services/knowledge_search.py`
- `services/ai-planning-service/app/services/llm_response_parser.py`
- `services/ai-planning-service/app/services/ollama_embedding_client.py`
- `services/ai-planning-service/app/services/ollama_itinerary_generator.py`
- `services/ai-planning-service/tests/test_ollama_generator.py`
- `services/trip-service/internal/domain/aggregate/itinerary.go`

## Audit Trail

- EXTRACTED: 525 (69%)
- INFERRED: 238 (31%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*