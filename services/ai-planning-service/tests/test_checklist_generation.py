import json

import pytest
from fastapi.testclient import TestClient

from app.application import create_app
from app.config import Settings
from app.services.llm_response_parser import LLMResponseParseError, parse_checklist_response


def test_generate_checklist_mock_uses_route_weather_and_activity_context() -> None:
    client = TestClient(create_app(Settings(itinerary_generator_mode="mock")))

    response = client.post("/generate-checklist", json=_payload())

    assert response.status_code == 200
    body = response.json()
    titles = {item["title"] for item in body["items"]}
    categories = {item["category"] for item in body["items"]}

    assert "Passport, ID, or required travel documents" in titles
    assert "Ferry or boat schedule verification" in titles
    assert "Hiking layers and trail-ready footwear" in titles
    assert "Rain jacket or compact umbrella" in titles
    assert "Sun protection and refillable water bottle" in titles
    assert {"documents", "transport", "camping_hiking", "weather"}.issubset(categories)
    assert body["warnings"]


def test_generate_checklist_category_mode_filters_categories() -> None:
    client = TestClient(create_app(Settings(itinerary_generator_mode="mock")))
    payload = _payload()
    payload["generationOptions"] = {
        "mode": "category",
        "categories": ["weather"],
        "preserveCheckedItems": True,
        "preserveManualItems": True,
        "replaceAiItems": False,
    }

    response = client.post("/generate-checklist", json=payload)

    assert response.status_code == 200
    body = response.json()
    assert body["items"]
    assert {item["category"] for item in body["items"]} == {"weather"}


def test_parse_checklist_response_accepts_strict_shape() -> None:
    response = parse_checklist_response(
        json.dumps(
            {
                "title": "Packing & preparation checklist",
                "summary": "Prepare for the trip.",
                "items": [
                    {
                        "title": "Passport",
                        "description": "Check document validity.",
                        "category": "documents",
                        "itemType": "document",
                        "priority": "critical",
                        "reason": "Required for travel.",
                        "metadata": {},
                    }
                ],
                "warnings": ["Verify travel requirements independently."],
            }
        )
    )

    assert response.items[0].category == "documents"
    assert response.items[0].item_type == "document"


def test_parse_checklist_response_rejects_extra_top_level_fields() -> None:
    with pytest.raises(LLMResponseParseError):
        parse_checklist_response(
            json.dumps(
                {
                    "title": "Checklist",
                    "summary": "Prepare.",
                    "items": [],
                    "warnings": [],
                    "extra": True,
                }
            )
        )


def _payload() -> dict[str, object]:
    return {
        "trip": {
            "id": "11111111-1111-1111-1111-111111111111",
            "title": "Alpine island trip",
            "destination": "Mallorca",
            "startDate": "2026-09-10",
            "durationDays": 5,
            "travelers": 2,
            "budget": {"amount": 1200, "currency": "EUR"},
            "interests": ["hiking", "nature"],
            "pace": "balanced",
            "tripType": "multi_destination",
        },
        "route": {
            "preferences": {
                "preferredModes": ["ferry"],
                "tripStyles": ["camping", "hiking"],
            },
            "legs": [
                {
                    "fromStopId": "origin",
                    "toStopId": "island",
                    "mode": "ferry",
                }
            ],
        },
        "weather": {
            "destination": "Mallorca",
            "days": [
                {
                    "date": "2026-09-10",
                    "condition": "rain",
                    "temperatureMinC": 18,
                    "temperatureMaxC": 31,
                    "precipitationChance": 70,
                    "windSpeedKph": 20,
                    "summary": "Rain and strong sun later",
                }
            ],
        },
        "itinerary": {
            "days": [
                {
                    "day": 1,
                    "title": "Trail and ferry day",
                    "items": [
                        {
                            "time": "09:00",
                            "type": "activity",
                            "name": "Coastal hike",
                            "note": "Trail with exposed sections",
                            "transportMode": "hiking",
                        },
                        {
                            "time": "15:00",
                            "type": "transport",
                            "name": "Ferry transfer",
                            "note": "Verify seasonal departures",
                            "transportMode": "ferry",
                        },
                    ],
                }
            ]
        },
        "generationOptions": {
            "mode": "full",
            "categories": [],
            "preserveCheckedItems": True,
            "preserveManualItems": True,
            "replaceAiItems": False,
        },
        "outputLanguage": "en",
    }
