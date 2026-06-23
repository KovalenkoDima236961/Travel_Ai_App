# MockItineraryGenerator

> God node · 31 connections · `services/ai-planning-service/app/services/itinerary_generator.py`

## Connections by Relation

### contains
- [[itinerary_generator.py]] `EXTRACTED`

### method
- [[._intensive_items()]] `EXTRACTED`
- [[._items_for_day()]] `EXTRACTED`
- [[._balanced_items()]] `EXTRACTED`
- [[._relaxed_items()]] `EXTRACTED`
- [[._morning_item()]] `EXTRACTED`
- [[._lunch_item()]] `EXTRACTED`
- [[._afternoon_item()]] `EXTRACTED`
- [[._personalized_interests()]] `EXTRACTED`
- [[.generate()]] `EXTRACTED`
- [[._title_for_day()]] `EXTRACTED`
- [[._evening_item()]] `EXTRACTED`
- [[.regenerate_day()]] `EXTRACTED`
- [[.regenerate_item()]] `EXTRACTED`
- [[._secondary_activity_name()]] `EXTRACTED`
- [[._secondary_activity_note()]] `EXTRACTED`

### uses
- [[OllamaItineraryGenerator]] `INFERRED`
- [[OllamaClientError]] `INFERRED`
- [[Raised when the Ollama API cannot provide a usable response.]] `INFERRED`
- [[GenerateItineraryRequest]] `INFERRED`
- [[ItineraryResponse]] `INFERRED`
- [[StaticDestinationKnowledgeProvider]] `INFERRED`
- [[StaticKnowledgeSearchService]] `INFERRED`
- [[FailingDestinationKnowledgeProvider]] `INFERRED`
- [[RegenerateDayRequest]] `INFERRED`
- [[RegenerateDayResponse]] `INFERRED`
- [[RegenerateItemResponse]] `INFERRED`
- [[FakeKnowledgeSearchService]] `INFERRED`
- [[RegenerateItemRequest]] `INFERRED`
- [[ItineraryItem]] `INFERRED`
- [[ItineraryDay]] `INFERRED`

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*