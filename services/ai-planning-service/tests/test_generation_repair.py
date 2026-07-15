from copy import deepcopy

from fastapi.testclient import TestClient

from app.application import create_app
from app.config import Settings
from app.schemas.generation_repair import RepairGenerationOutputRequest
from app.services.prompt_builder import build_generation_output_repair_prompt

client = TestClient(create_app(Settings(itinerary_generator_mode="mock")))


def repair_payload() -> dict:
    return {
        "generationType": "full_itinerary",
        "currentOutput": {
            "destination": "Vienna",
            "currency": "EUR",
            "days": [
                {
                    "day": 1,
                    "title": "Arrival",
                    "primaryStopId": "stop_1",
                    "items": [
                        {
                            "time": "09:00",
                            "endTime": "10:00",
                            "type": "activity",
                            "name": "Old town walk",
                            "estimatedCost": {
                                "amount": 12,
                                "currency": "EUR",
                                "category": "activity",
                            },
                        }
                    ],
                }
            ],
        },
        "validationIssues": [
            {
                "id": "activity_before_transport_arrival:day_1:item_0:leg_1",
                "category": "transport",
                "severity": "critical",
                "title": "Activity starts before transport arrival",
                "fixability": "fixable_by_ai",
                "dayNumber": 1,
                "itemIndex": 0,
                "routeLegId": "leg_1",
            },
            {
                "id": "transfer_item_missing_or_mismatch:leg_1",
                "category": "transport",
                "severity": "high",
                "title": "Selected transport is missing from itinerary",
                "fixability": "fixable_by_ai",
                "dayNumber": 1,
                "routeLegId": "leg_1",
            },
        ],
        "planningContext": {
            "trip": {
                "Destination": "Vienna",
                "Days": 1,
                "BudgetCurrency": "EUR",
            },
            "route": {
                "stops": [{"id": "stop_1", "destination": "Vienna"}],
                "legs": [
                    {
                        "id": "leg_1",
                        "fromStopId": "origin",
                        "toStopId": "stop_1",
                        "fromName": "Bratislava",
                        "toName": "Vienna",
                        "mode": "train",
                        "estimatedDurationMinutes": 60,
                        "selectedTransportOption": {
                            "id": "opt_1",
                            "mode": "train",
                            "provider": "mock",
                            "departureDate": "2026-09-10",
                            "departureTime": "08:00",
                            "arrivalDate": "2026-09-10",
                            "arrivalTime": "11:00",
                            "durationMinutes": 180,
                            "estimatedPrice": {"amount": 18, "currency": "EUR"},
                        },
                    }
                ],
            },
        },
        "repairScope": {"type": "full_output"},
        "constraints": {
            "preserveUnaffectedDays": True,
            "preserveUserEditedItems": True,
            "outputLanguage": "en",
        },
    }


def test_repair_generation_output_moves_activity_and_adds_transfer() -> None:
    response = client.post("/repair-generation-output", json=repair_payload())

    assert response.status_code == 200
    body = response.json()
    assert "repairedOutput" in body
    items = body["repairedOutput"]["days"][0]["items"]
    assert items[0]["type"] == "transport"
    assert items[0]["transfer"]["legId"] == "leg_1"
    assert items[1]["name"] == "Old town walk"
    assert items[1]["time"] == "11:30"
    assert {change["type"] for change in body["changesMade"]} == {
        "item_moved",
        "transfer_item_added",
    }


def test_repair_generation_output_normalizes_missing_day() -> None:
    payload = deepcopy(repair_payload())
    payload["planningContext"]["trip"]["Days"] = 2
    payload["validationIssues"] = [
        {
            "id": "itinerary_day_count_mismatch",
            "category": "itinerary",
            "severity": "critical",
            "title": "Day count does not match trip duration",
            "fixability": "fixable_by_ai",
        }
    ]

    response = client.post("/repair-generation-output", json=payload)

    assert response.status_code == 200
    days = response.json()["repairedOutput"]["days"]
    assert [day["day"] for day in days] == [1, 2]
    assert days[1]["items"][0]["name"] == "Flexible itinerary block"


def test_generation_output_repair_prompt_contains_contract() -> None:
    request = RepairGenerationOutputRequest.model_validate(repair_payload())

    prompt = build_generation_output_repair_prompt(request)

    assert "repairedOutput" in prompt
    assert "changesMade" in prompt
    assert "transfer.legId" in prompt
    assert "activity_before_transport_arrival" in prompt
