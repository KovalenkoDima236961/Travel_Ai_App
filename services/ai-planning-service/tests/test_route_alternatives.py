from copy import deepcopy

from fastapi.testclient import TestClient

from app.application import create_app
from app.config import Settings
from app.schemas.route_alternatives import RouteAlternativeRequest
from app.services.prompt_builder import build_route_alternatives_prompt

client = TestClient(create_app(Settings(itinerary_generator_mode="mock")))


def payload() -> dict:
    return {
        "origin": {
            "name": "Bratislava",
            "country": "Slovakia",
            "coordinates": {"lat": 48.1486, "lng": 17.1077},
        },
        "prompt": "A 5-day Austria trip with nature, old towns, and train travel.",
        "durationDays": 5,
        "startDate": "2026-09-10",
        "budget": {"amount": 700, "currency": "EUR"},
        "travelers": 2,
        "outputLanguage": "en",
        "planningConstraints": {
            "source": "route_alternatives",
            "language": "en",
            "transport": {"preferredModes": ["train"], "avoidModes": ["flight"]},
            "tripStyles": ["nature", "culture", "train_trip"],
        },
        "suggestionCount": 3,
    }


def test_mock_austria_prompt_returns_three_route_alternatives() -> None:
    response = client.post("/suggest-route-alternatives", json=payload())

    assert response.status_code == 200
    body = response.json()
    assert len(body["alternatives"]) == 3
    assert [item["id"] for item in body["alternatives"]] == [
        "classic-austria-train-route",
        "relaxed-two-city-route",
        "nature-heavy-route",
    ]
    first = body["alternatives"][0]
    assert [stop["destination"] for stop in first["route"]["stops"]] == [
        "Vienna",
        "Salzburg",
        "Hallstatt",
    ]
    assert first["difficulty"] == "balanced"
    assert 0 <= first["scores"]["overallFit"] <= 100


def test_avoid_flight_and_preferred_train_use_train_legs() -> None:
    response = client.post("/suggest-route-alternatives", json=payload())

    assert response.status_code == 200
    body = response.json()
    for alternative in body["alternatives"]:
        modes = [leg["mode"] for leg in alternative["route"]["legs"]]
        assert "flight" not in modes
        assert set(modes) == {"train"}


def test_road_trip_with_car_available_includes_car_route() -> None:
    body = deepcopy(payload())
    body["planningConstraints"]["transport"] = {
        "preferredModes": ["car"],
        "avoidModes": ["flight"],
        "carAvailable": True,
    }
    body["planningConstraints"]["tripStyles"] = ["road_trip", "nature"]

    response = client.post("/suggest-route-alternatives", json=body)

    assert response.status_code == 200
    modes = {
        leg["mode"]
        for alternative in response.json()["alternatives"]
        for leg in alternative["route"]["legs"]
    }
    assert "rental_car" in modes or "car" in modes
    assert "flight" not in modes


def test_ukrainian_localizes_user_facing_text_but_keeps_keys() -> None:
    body = payload()
    body["outputLanguage"] = "uk"
    body["planningConstraints"]["language"] = "uk"

    response = client.post("/suggest-route-alternatives", json=body)

    assert response.status_code == 200
    result = response.json()
    assert "sessionTitle" in result
    assert "Варіанти" in result["sessionTitle"]
    assert "scores" in result["alternatives"][0]
    assert "overallFit" in result["alternatives"][0]["scores"]


def test_prompt_builder_contains_route_alternative_rules() -> None:
    request = RouteAlternativeRequest.model_validate(payload())

    prompt = build_route_alternatives_prompt(request)

    assert "Generate route alternatives, not a detailed day-by-day itinerary." in prompt
    assert "do not claim live schedules" in prompt
    assert "Bratislava" in prompt
    assert "route_alternatives" in prompt
