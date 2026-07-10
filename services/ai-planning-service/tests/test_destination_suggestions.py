from copy import deepcopy

from fastapi.testclient import TestClient

from app.application import create_app
from app.config import Settings
from app.schemas.destination_suggestion import DestinationSuggestionRequest
from app.services.prompt_builder import build_destination_suggestion_prompt

client = TestClient(create_app(Settings(itinerary_generator_mode="mock")))


def payload() -> dict:
    return {
        "prompt": "A cheap warm food weekend",
        "mode": "prompt",
        "outputLanguage": "en",
        "userContext": {
            "homeCity": "Bratislava",
            "homeCountry": "Slovakia",
            "preferredCurrency": "EUR",
            "preferredLanguage": "en",
            "preferences": {"travelStyles": ["food"], "avoid": ["nightclubs"]},
        },
        "tripContext": {
            "durationDays": 3,
            "budget": {"amount": 700, "currency": "EUR"},
            "travelers": 2,
            "scope": "personal",
        },
        "constraints": {"suggestionCount": 5, "avoidPreviouslyVisited": True},
    }


def test_prompt_mode_returns_deterministic_suggestions() -> None:
    response = client.post("/suggest-destinations", json=payload())

    assert response.status_code == 200
    body = response.json()
    assert 3 <= len(body["suggestions"]) <= 5
    assert body["suggestions"][0]["city"] == "Kraków"
    assert 0 <= body["suggestions"][0]["matchScore"] <= 100


def test_surprise_mode_avoids_previous_destination() -> None:
    body = payload()
    body["prompt"] = ""
    body["mode"] = "surprise"
    body["previousTrips"] = [
        {"destination": "Prague", "durationDays": 3, "createdAt": "2026-05-12"}
    ]

    response = client.post("/suggest-destinations", json=body)

    assert response.status_code == 200
    assert all(item["city"] != "Prague" for item in response.json()["suggestions"])
    assert response.json()["suggestions"][0]["city"] == "Vienna"


def test_refine_mode_changes_to_nature_alternatives() -> None:
    body = payload()
    body["prompt"] = ""
    body["mode"] = "refine"
    body["refinement"] = {
        "instruction": "Cheaper and more nature",
        "previousSuggestions": [],
    }

    response = client.post("/suggest-destinations", json=body)

    assert response.status_code == 200
    cities = [item["city"] for item in response.json()["suggestions"]]
    assert cities[:3] == ["Brno", "Kraków", "Budapest"]


def test_ukrainian_localizes_user_facing_text_but_keeps_keys() -> None:
    body = payload()
    body["outputLanguage"] = "uk"

    response = client.post("/suggest-destinations", json=body)

    assert response.status_code == 200
    result = response.json()
    assert "suggestions" in result
    assert "whyItFits" in result["suggestions"][0]
    assert "відповідає" in result["suggestions"][0]["whyItFits"]


def test_invalid_mode_and_language_are_rejected() -> None:
    body = payload()
    body["mode"] = "random"
    assert client.post("/suggest-destinations", json=body).status_code == 422

    body = payload()
    body["outputLanguage"] = "de"
    assert client.post("/suggest-destinations", json=body).status_code == 422


def test_prompt_builder_contains_sanitized_context() -> None:
    body = deepcopy(payload())
    body["previousTrips"] = [{"destination": "Prague", "durationDays": 3}]
    request = DestinationSuggestionRequest.model_validate(body)

    prompt = build_destination_suggestion_prompt(request)

    assert "Bratislava" in prompt
    assert "Prague" in prompt
    assert "nightclubs" in prompt
    assert "comments" not in prompt
    assert "shareToken" not in prompt
