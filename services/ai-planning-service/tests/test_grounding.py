from uuid import UUID

from app.schemas.grounding import GroundingContext
from app.schemas.itinerary import GenerateItineraryRequest
from app.services.itinerary_generator import MockItineraryGenerator
from app.services.prompt_builder import build_itinerary_prompt


def _request() -> GenerateItineraryRequest:
    return GenerateItineraryRequest(
        tripId=UUID("00000000-0000-0000-0000-000000000001"),
        destination="Rome",
        days=2,
        travelers=2,
        interests=["food", "culture"],
        groundingContext={
            "status": "available",
            "destination": {"canonicalName": "Rome", "countryCode": "IT"},
            "places": [
                {
                    "id": "rome-colosseum",
                    "canonicalName": "Colosseum",
                    "category": "landmark",
                    "typicalDurationMinutes": 120,
                    "outdoor": True,
                    "rainFriendly": False,
                    "confidence": 0.95,
                },
                {
                    "id": "rome-pantheon",
                    "canonicalName": "Pantheon",
                    "category": "landmark",
                    "typicalDurationMinutes": 45,
                    "outdoor": False,
                    "rainFriendly": True,
                    "confidence": 0.95,
                },
            ],
        },
    )


def test_grounding_context_deduplicates_places() -> None:
    context = GroundingContext.model_validate(
        {
            "status": "available",
            "places": [
                {"id": "a", "canonicalName": "Pantheon", "category": "landmark", "confidence": 0.9},
                {"id": "a", "canonicalName": "Pantheon", "category": "landmark", "confidence": 0.8},
            ],
        }
    )
    assert len(context.places) == 1


def test_mock_generation_uses_only_grounded_place_names() -> None:
    response = MockItineraryGenerator().generate(_request())
    items = [item for day in response.days for item in day.items]
    assert {item.name for item in items} <= {"Colosseum", "Pantheon"}
    assert all(item.grounding_source == "grounded" for item in items)
    assert all(item.needs_place_review is False for item in items)


def test_prompt_includes_grounding_rules_and_compact_context() -> None:
    prompt = build_itinerary_prompt(_request())
    assert "GROUNDING CONTEXT:" in prompt
    assert "name=Colosseum" in prompt
    assert "Do not invent a specific place name" in prompt
