from copy import deepcopy

from fastapi.testclient import TestClient

from app.main import app, create_app
from app.schemas.itinerary import GenerateItineraryRequest

client = TestClient(app)


VALID_PAYLOAD = {
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 4,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history", "hidden_gems"],
    "pace": "balanced",
}

CURRENT_ITINERARY = {
    "days": [
        {
            "day": 1,
            "title": "Arrival",
            "items": [
                {
                    "time": "09:00",
                    "type": "activity",
                    "name": "Original walk",
                    "note": "Keep this item.",
                    "estimatedCost": 0,
                }
            ],
        },
        {
            "day": 2,
            "title": "Museums",
            "items": [
                {
                    "time": "10:00",
                    "type": "place",
                    "name": "Original museum",
                    "note": "Candidate for replacement.",
                    "estimatedCost": 18,
                },
                {
                    "time": "13:00",
                    "type": "food",
                    "name": "Original lunch",
                    "note": "Candidate for replacement.",
                    "estimatedCost": 20,
                },
            ],
        },
    ]
}


def partial_payload() -> dict:
    return {
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
        "currentItinerary": deepcopy(CURRENT_ITINERARY),
        "dayNumber": 2,
        "instruction": "make this cheaper and more relaxed",
    }


def test_health_endpoint_returns_ok() -> None:
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok", "service": "ai-planning-service"}


def test_version_endpoint_returns_safe_build_metadata(monkeypatch) -> None:
    for name in ("APP_VERSION", "GIT_SHA", "BUILD_TIME", "APP_ENV"):
        monkeypatch.delenv(name, raising=False)
    response = client.get("/version")

    assert response.status_code == 200
    assert response.json() == {
        "service": "ai-planning-service",
        "version": "dev",
        "gitSha": "unknown",
        "buildTime": "unknown",
        "environment": "local",
        "apiContractVersion": "dev",
    }


def test_ready_endpoint_returns_ready_in_mock_mode() -> None:
    test_app = create_app()
    test_client = TestClient(test_app)

    response = test_client.get("/ready")

    assert response.status_code == 200
    assert response.json() == {
        "status": "ready",
        "service": "ai-planning-service",
        "dependencies": {"app": "ok"},
        "checks": {"app": "ok"},
    }


def test_generate_itinerary_success() -> None:
    response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    assert "days" in body
    assert body["days"][0]["day"] == 1
    assert len(body["days"][0]["items"]) >= 3


def test_generate_itinerary_accepts_optional_user_context() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["userProfile"] = {
        "userId": "550e8400-e29b-41d4-a716-446655440000",
        "displayName": "Test Traveler",
        "homeCity": "Bratislava",
        "homeCountry": "Slovakia",
        "preferredCurrency": "EUR",
        "preferredLanguage": "en",
    }
    payload["userPreferences"] = {
        "userId": "550e8400-e29b-41d4-a716-446655440000",
        "travelStyles": ["budget", "food", "hidden_gems"],
        "pace": "balanced",
        "maxWalkingKmPerDay": 8,
        "foodPreferences": ["local", "cheap"],
        "avoid": ["nightclubs"],
        "preferredTransport": ["walking", "public_transport"],
        "accommodationStyle": ["budget_hotel"],
        "dietaryRestrictions": [],
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    assert len(response.json()["days"]) == VALID_PAYLOAD["days"]


def test_generate_itinerary_accepts_optional_weather_forecast() -> None:
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
            for _ in range(VALID_PAYLOAD["days"])
        ],
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    request = GenerateItineraryRequest.model_validate(payload)
    assert request.weather_forecast is not None
    assert request.weather_forecast.days[0].precipitation_chance == 5


def test_generate_itinerary_accepts_optional_accommodation() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["accommodation"] = {
        "name": "Hotel Roma",
        "type": "Hotel",
        "address": "Via Roma 10",
        "place": {
            "provider": "google",
            "providerPlaceId": "hotel-roma",
            "name": "Hotel Roma",
            "address": "Via Roma 10",
            "latitude": 41.9028,
            "longitude": 12.4964,
            "mapUrl": "https://maps.example/hotel-roma",
            "category": "lodging",
        },
        "checkInDate": "2026-08-10",
        "checkOutDate": "2026-08-14",
        "estimatedCost": {"amount": 420, "currency": "EUR", "category": "accommodation"},
        "notes": "Near the train station.",
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    request = GenerateItineraryRequest.model_validate(payload)
    assert request.accommodation is not None
    assert request.accommodation.name == "Hotel Roma"
    assert request.accommodation.type == "hotel"
    assert request.accommodation.place is not None
    assert request.accommodation.place.latitude == 41.9028


def test_user_preferences_arrays_are_trimmed_deduplicated_and_optional() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["userPreferences"] = {
        "travelStyles": [" budget ", "", "budget", "hidden_gems"],
        "foodPreferences": [" local ", None, "local", "cheap"],
        "avoid": [" nightclubs ", "nightclubs"],
        "preferredTransport": None,
        "accommodationStyle": [" budget_hotel "],
        "dietaryRestrictions": [],
    }

    request = GenerateItineraryRequest.model_validate(payload)

    assert request.user_preferences is not None
    assert request.user_preferences.travel_styles == ["budget", "hidden_gems"]
    assert request.user_preferences.food_preferences == ["local", "cheap"]
    assert request.user_preferences.avoid == ["nightclubs"]
    assert request.user_preferences.preferred_transport == []


def test_generated_itinerary_has_requested_number_of_days() -> None:
    response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 200
    assert len(response.json()["days"]) == VALID_PAYLOAD["days"]


def test_generated_itinerary_includes_destination_in_title_or_note() -> None:
    response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    text_values = []
    for day in body["days"]:
        text_values.append(day["title"])
        text_values.extend(item.get("note") or "" for item in day["items"])

    assert any("Rome" in text for text in text_values)


def test_missing_destination_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload.pop("destination")

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_empty_destination_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["destination"] = " "

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_invalid_trip_id_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["tripId"] = "not-a-uuid"

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_days_less_than_one_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["days"] = 0

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_days_greater_than_thirty_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["days"] = 31

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_travelers_less_than_one_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["travelers"] = 0

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_negative_budget_amount_returns_validation_error() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["budgetAmount"] = -1

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 422


def test_unexpected_generator_error_returns_clean_generation_error() -> None:
    class BrokenGenerator:
        def generate(self, request: GenerateItineraryRequest) -> None:
            raise RuntimeError("internal failure")

    test_app = create_app()
    test_app.state.itinerary_generator = BrokenGenerator()
    test_client = TestClient(test_app)

    response = test_client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 500
    assert response.json() == {"error": "Failed to generate itinerary"}


def test_empty_budget_currency_defaults_to_eur() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["budgetCurrency"] = ""

    request = GenerateItineraryRequest.model_validate(payload)

    assert request.budget_currency == "EUR"


def test_relaxed_pace_produces_fewer_or_equal_items_than_intensive_pace() -> None:
    relaxed_payload = deepcopy(VALID_PAYLOAD)
    relaxed_payload["pace"] = "relaxed"
    intensive_payload = deepcopy(VALID_PAYLOAD)
    intensive_payload["pace"] = "intensive"

    relaxed_response = client.post("/generate-itinerary", json=relaxed_payload)
    intensive_response = client.post("/generate-itinerary", json=intensive_payload)

    assert relaxed_response.status_code == 200
    assert intensive_response.status_code == 200
    relaxed_items = relaxed_response.json()["days"][0]["items"]
    intensive_items = intensive_response.json()["days"][0]["items"]
    assert len(relaxed_items) <= len(intensive_items)


def test_mock_generator_uses_hidden_gems_and_local_food_preferences() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["interests"] = ["history"]
    payload["userPreferences"] = {
        "travelStyles": ["hidden_gems"],
        "foodPreferences": ["local"],
        "avoid": ["nightclubs"],
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    body = response.json()
    text = " ".join(
        " ".join(
            [
                day["title"],
                *[item["name"] + " " + (item.get("note") or "") for item in day["items"]],
            ]
        )
        for day in body["days"]
    ).casefold()
    assert "hidden-gem" in text
    assert "local neighborhood lunch" in text
    assert "nightclub" not in text


def test_mock_generator_adapts_to_rainy_weather() -> None:
    payload = deepcopy(VALID_PAYLOAD)
    payload["weatherForecast"] = {
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
            for _ in range(VALID_PAYLOAD["days"])
        ],
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    first_item = response.json()["days"][0]["items"][0]
    assert "Indoor" in first_item["name"]
    assert "Rain is likely" in first_item["note"]


def test_mock_generator_adapts_to_hot_weather() -> None:
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
            for _ in range(VALID_PAYLOAD["days"])
        ],
    }

    response = client.post("/generate-itinerary", json=payload)

    assert response.status_code == 200
    body = response.json()
    text = " ".join(item.get("note") or "" for day in body["days"] for item in day["items"])
    assert "High heat expected" in text


def test_regenerate_day_success_returns_replacement_day_only() -> None:
    response = client.post("/regenerate-day", json=partial_payload())

    assert response.status_code == 200
    body = response.json()
    assert set(body.keys()) == {"day", "metadata"}
    assert body["metadata"]["promptVersion"] == "day_regeneration_v1"
    assert body["day"]["day"] == 2
    assert body["day"]["title"]
    assert len(body["day"]["items"]) >= 1
    assert "days" not in body


def test_regenerate_item_success_returns_replacement_item_only() -> None:
    payload = partial_payload()
    payload["itemIndex"] = 1

    response = client.post("/regenerate-item", json=payload)

    assert response.status_code == 200
    body = response.json()
    assert set(body.keys()) == {"item", "metadata"}
    assert body["metadata"]["promptVersion"] == "item_regeneration_v1"
    assert body["item"]["time"]
    assert body["item"]["type"]
    assert body["item"]["name"]
    assert "day" not in body


def test_regenerate_day_adapts_to_rainy_weather() -> None:
    payload = partial_payload()
    payload["weatherForecast"] = {
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
            for _ in range(VALID_PAYLOAD["days"])
        ],
    }

    response = client.post("/regenerate-day", json=payload)

    assert response.status_code == 200
    text = " ".join(item.get("note") or "" for item in response.json()["day"]["items"])
    assert "Rain is likely" in text


def test_regenerate_item_adapts_to_hot_weather() -> None:
    payload = partial_payload()
    payload["itemIndex"] = 1
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
            for _ in range(VALID_PAYLOAD["days"])
        ],
    }

    response = client.post("/regenerate-item", json=payload)

    assert response.status_code == 200
    assert "High heat expected" in response.json()["item"]["note"]


def test_regenerate_day_invalid_day_number_returns_400() -> None:
    payload = partial_payload()
    payload["dayNumber"] = 9

    response = client.post("/regenerate-day", json=payload)

    assert response.status_code == 400
    assert "dayNumber" in response.json()["error"]


def test_regenerate_item_invalid_item_index_returns_400() -> None:
    payload = partial_payload()
    payload["itemIndex"] = 9

    response = client.post("/regenerate-item", json=payload)

    assert response.status_code == 400
    assert "itemIndex" in response.json()["error"]


def test_regenerate_day_instruction_too_long_returns_400() -> None:
    payload = partial_payload()
    payload["instruction"] = "x" * 501

    response = client.post("/regenerate-day", json=payload)

    assert response.status_code == 400
    assert "500" in response.json()["error"]


def test_regenerate_item_accepts_optional_user_context() -> None:
    payload = partial_payload()
    payload["itemIndex"] = 0
    payload["userPreferences"] = {
        "travelStyles": ["budget", "food"],
        "foodPreferences": ["local", "cheap"],
        "avoid": ["museums"],
    }

    response = client.post("/regenerate-item", json=payload)

    assert response.status_code == 200
    body = response.json()
    assert body["item"]["estimatedCost"]["amount"] <= 15
