from copy import deepcopy

from app.schemas.itinerary import GenerateItineraryRequest, RegenerateDayRequest
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.prompt_builder import (
    build_itinerary_prompt,
    build_regenerate_day_prompt,
    build_repair_prompt,
)

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


def test_itinerary_prompt_includes_user_context_when_provided() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["userProfile"] = {
        "homeCity": "Bratislava",
        "homeCountry": "Slovakia",
        "preferredCurrency": "EUR",
        "preferredLanguage": "en",
    }
    payload["userPreferences"] = {
        "travelStyles": ["budget", "food", "hidden_gems"],
        "pace": "balanced",
        "maxWalkingKmPerDay": 8,
        "foodPreferences": ["local", "cheap"],
        "avoid": ["nightclubs"],
        "preferredTransport": ["walking", "public_transport"],
        "accommodationStyle": ["budget_hotel"],
        "dietaryRestrictions": [],
    }

    prompt = build_itinerary_prompt(GenerateItineraryRequest.model_validate(payload))

    assert "USER PROFILE:" in prompt
    assert "- Home city: Bratislava" in prompt
    assert "USER TRAVEL PREFERENCES:" in prompt
    assert "- Travel styles: budget, food, hidden gems" in prompt
    assert "- Avoid: nightclubs" in prompt
    assert "Avoid preference items listed under Avoid" in prompt


def test_itinerary_prompt_omits_user_context_when_not_provided() -> None:
    prompt = build_itinerary_prompt(_request())

    assert "USER PROFILE:" not in prompt
    assert "USER TRAVEL PREFERENCES:" not in prompt


def test_itinerary_prompt_includes_weather_forecast_and_warnings() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["weatherForecast"] = {
        "destination": "Rome",
        "provider": "mock",
        "days": [
            {
                "date": "2026-08-10",
                "condition": "hot",
                "temperatureMinC": 24,
                "temperatureMaxC": 35,
                "precipitationChance": 5,
                "windSpeedKph": 10,
                "summary": "Hot and sunny",
                "warnings": ["High heat: avoid long outdoor walks at midday"],
            }
        ],
    }

    prompt = build_itinerary_prompt(GenerateItineraryRequest.model_validate(payload))

    assert "WEATHER FORECAST:" in prompt
    assert "2026-08-10: Hot and sunny" in prompt
    assert "Warnings: High heat: avoid long outdoor walks at midday" in prompt
    assert "Avoid long outdoor walks during high heat." in prompt


def test_itinerary_prompt_omits_weather_section_when_not_provided() -> None:
    prompt = build_itinerary_prompt(_request())

    assert "WEATHER FORECAST:" not in prompt


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


def test_repair_prompt_includes_user_context() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["userPreferences"] = {
        "travelStyles": ["hidden_gems"],
        "avoid": ["nightclubs"],
        "foodPreferences": ["local"],
    }
    request = GenerateItineraryRequest.model_validate(payload)

    prompt = build_repair_prompt(
        request=request,
        invalid_response_text='{"days": []}',
        validation_error="missing days",
    )

    assert "USER TRAVEL PREFERENCES:" in prompt
    assert "- Avoid: nightclubs" in prompt
    assert "Preserve personalization from the user profile and travel preferences" in prompt


def test_repair_prompt_preserves_weather_context() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["weatherForecast"] = {
        "destination": "Rome",
        "provider": "mock",
        "days": [
            {
                "date": "2026-08-10",
                "condition": "light_rain",
                "temperatureMinC": 20,
                "temperatureMaxC": 26,
                "precipitationChance": 70,
                "windSpeedKph": 18,
                "summary": "Light rain likely",
                "warnings": ["Rain likely: consider indoor alternatives"],
            }
        ],
    }
    request = GenerateItineraryRequest.model_validate(payload)

    prompt = build_repair_prompt(
        request=request,
        invalid_response_text='{"days": []}',
        validation_error="missing days",
    )

    assert "WEATHER FORECAST:" in prompt
    assert "Rain likely: consider indoor alternatives" in prompt
    assert "Preserve weather-aware choices from the weather forecast" in prompt


def test_partial_regeneration_prompt_includes_weather_context() -> None:
    payload = {
        "trip": {
            "id": VALID_PAYLOAD["tripId"],
            "destination": VALID_PAYLOAD["destination"],
            "startDate": VALID_PAYLOAD["startDate"],
            "days": VALID_PAYLOAD["days"],
            "budgetAmount": VALID_PAYLOAD["budgetAmount"],
            "budgetCurrency": VALID_PAYLOAD["budgetCurrency"],
            "travelers": VALID_PAYLOAD["travelers"],
            "interests": VALID_PAYLOAD["interests"],
            "pace": VALID_PAYLOAD["pace"],
        },
        "currentItinerary": {
            "days": [
                {
                    "day": 1,
                    "title": "Original day",
                    "items": [
                        {
                            "time": "09:00",
                            "type": "activity",
                            "name": "Original walk",
                            "note": "Outdoor route",
                            "estimatedCost": 0,
                        }
                    ],
                }
            ]
        },
        "dayNumber": 1,
        "weatherForecast": {
            "destination": "Rome",
            "provider": "mock",
            "days": [
                {
                    "date": "2026-08-10",
                    "condition": "rain",
                    "temperatureMinC": 20,
                    "temperatureMaxC": 25,
                    "precipitationChance": 80,
                    "windSpeedKph": 14,
                    "summary": "Rain likely",
                    "warnings": ["Rain likely: consider indoor alternatives"],
                }
            ],
        },
    }

    prompt = build_regenerate_day_prompt(RegenerateDayRequest.model_validate(payload))

    assert "WEATHER FORECAST:" in prompt
    assert "Rain likely: consider indoor alternatives" in prompt
    assert "Adapt the replacement day to the weather forecast when relevant." in prompt
