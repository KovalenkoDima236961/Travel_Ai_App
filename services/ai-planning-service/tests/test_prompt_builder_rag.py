from copy import deepcopy

from app.schemas.itinerary import GenerateItineraryRequest
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.prompt_builder import build_itinerary_prompt, build_repair_prompt

VALID_PAYLOAD = {
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 2,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history"],
    "pace": "balanced",
}


def _request() -> GenerateItineraryRequest:
    return GenerateItineraryRequest.model_validate(deepcopy(VALID_PAYLOAD))


def _rag_chunks() -> list[KnowledgeSearchResult]:
    return [
        KnowledgeSearchResult(
            id="rome:food.md:0",
            destination="rome",
            source="food.md",
            content="Avoid restaurants directly beside the Colosseum.",
            score=0.9,
            metadata={"chunkIndex": 0},
        )
    ]


def test_itinerary_prompt_includes_rag_context_when_chunks_exist() -> None:
    prompt = build_itinerary_prompt(_request(), rag_chunks=_rag_chunks())

    assert "RAG CONTEXT:" in prompt
    assert "- Source: food.md" in prompt
    assert "Avoid restaurants directly beside the Colosseum." in prompt


def test_itinerary_prompt_preserves_json_schema_instructions_with_rag() -> None:
    prompt = build_itinerary_prompt(_request(), rag_chunks=_rag_chunks())

    assert "Return ONLY valid JSON" in prompt
    assert "The JSON must exactly match this schema" in prompt
    assert '"days"' in prompt
    assert "Do not include fields outside the schema." in prompt


def test_itinerary_prompt_works_without_rag_chunks() -> None:
    prompt = build_itinerary_prompt(_request(), rag_chunks=None)

    assert "RAG CONTEXT:" not in prompt
    assert "Return ONLY valid JSON" in prompt


def test_repair_prompt_includes_rag_context() -> None:
    prompt = build_repair_prompt(
        request=_request(),
        invalid_response_text='{"days": []}',
        validation_error="missing days",
        rag_chunks=_rag_chunks(),
    )

    assert "RAG CONTEXT:" in prompt
    assert "- Source: food.md" in prompt
    assert "The corrected JSON should still use the RAG context where relevant." in prompt
