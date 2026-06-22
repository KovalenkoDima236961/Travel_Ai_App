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


def test_health_endpoint_returns_ok() -> None:
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok", "service": "ai-planning-service"}


def test_generate_itinerary_success() -> None:
    response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    assert "days" in body
    assert body["days"][0]["day"] == 1
    assert len(body["days"][0]["items"]) >= 3


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
