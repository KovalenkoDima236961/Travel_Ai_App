# ruff: noqa: E501
from copy import deepcopy

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def payload() -> dict:
    return {
        "language": "en",
        "style": "concise",
        "includeLearningCandidates": True,
        "sourceSummary": {
            "trip": {
                "title": "Austria Trip",
                "destination": "Austria",
                "durationDays": 6,
                "tripType": "multi_destination",
            },
            "itineraryOutcome": {
                "plannedItemCount": 10,
                "doneItemCount": 8,
                "skippedItemCount": 1,
                "delayedItemCount": 1,
                "unknownItemCount": 0,
                "topCompletedItems": ["Museum walk"],
                "topSkippedItems": ["Ferry stop"],
            },
            "budgetOutcome": {
                "plannedTotal": {"amount": 800, "currency": "EUR"},
                "actualTotal": {"amount": 720, "currency": "EUR"},
                "variance": {"amount": -80, "currency": "EUR"},
                "receiptCoveragePercent": 65,
                "topCategories": [],
            },
            "routeOutcome": {
                "stops": ["Vienna"],
                "transportModes": ["train"],
                "verifiedTransportCount": 1,
                "unverifiedTransportCount": 0,
                "issues": [],
            },
            "checklistOutcome": {
                "completedChecklistItems": 8,
                "totalChecklistItems": 10,
                "completedReminders": 2,
                "totalReminders": 3,
            },
            "verificationOutcome": {
                "score": 80,
                "summary": "Most data was verified.",
                "verifiedCount": 4,
                "staleCount": 0,
                "missingCount": 1,
                "issues": [],
            },
            "learningCandidates": [],
        },
    }


def test_generate_trip_recap_returns_strict_editable_recap() -> None:
    response = client.post("/generate-trip-recap", json=payload())

    assert response.status_code == 200
    body = response.json()
    assert set(body) == {"recap", "warnings", "assumptions"}
    recap = body["recap"]
    assert recap["schemaVersion"] == "trip_recap_v1"
    assert recap["plannedVsActual"]["doneItemCount"] == 8
    assert recap["budget"]["actualTotal"] == {"amount": 720.0, "currency": "EUR"}
    assert recap["futurePreferences"][0]["approved"] is False


def test_generate_trip_recap_does_not_accept_raw_receipt_or_calendar_data() -> None:
    invalid = deepcopy(payload())
    invalid["sourceSummary"]["rawText"] = "receipt OCR text"

    response = client.post("/generate-trip-recap", json=invalid)

    assert response.status_code == 422
