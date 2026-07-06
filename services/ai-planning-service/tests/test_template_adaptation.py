from copy import deepcopy

from fastapi.testclient import TestClient

from app.main import app
from app.schemas.template_adaptation import TemplateAdaptationRequest
from app.services.prompt_builder import build_template_adaptation_prompt

client = TestClient(app)


TEMPLATE = {
    "schemaVersion": 1,
    "durationDays": 3,
    "days": [
        {
            "dayOffset": 0,
            "title": "Old Town",
            "items": [
                {
                    "name": "Prague Old Town walk",
                    "type": "activity",
                    "startTime": "09:00",
                    "endTime": "11:00",
                    "estimatedCost": {"amount": 0, "currency": "EUR"},
                    "notes": "Morning orientation walk.",
                },
                {
                    "name": "Local lunch",
                    "type": "food",
                    "startTime": "12:30",
                    "estimatedCost": {"amount": 15, "currency": "EUR"},
                },
            ],
        },
        {
            "dayOffset": 1,
            "title": "Museums",
            "items": [
                {
                    "name": "National Museum",
                    "type": "place",
                    "startTime": "10:00",
                    "estimatedCost": {"amount": 18, "currency": "EUR"},
                }
            ],
        },
        {
            "dayOffset": 2,
            "title": "Views",
            "items": [
                {"name": "Castle hill", "type": "place", "startTime": "10:00"},
            ],
        },
    ],
}

TARGET = {
    "destination": "Vienna",
    "startDate": "2026-09-10",
    "durationDays": 3,
    "budget": {"amount": 700, "currency": "EUR"},
    "travelers": 2,
    "pace": "balanced",
    "interests": ["museums", "food", "architecture"],
    "avoid": ["nightclubs"],
}

CONSTRAINTS = {
    "preserveStructure": True,
    "adaptCosts": True,
    "preserveMealStructure": True,
    "preserveActivityDensity": True,
    "specialInstructions": "Make it suitable for first-time visitors.",
}


def _payload(**overrides):
    payload = {
        "template": deepcopy(TEMPLATE),
        "target": deepcopy(TARGET),
        "constraints": deepcopy(CONSTRAINTS),
    }
    for key, value in overrides.items():
        payload[key] = value
    return payload


def test_mock_adaptation_same_duration_preserves_structure():
    payload = _payload()
    response = client.post("/adapt-template", json=payload)
    assert response.status_code == 200
    body = response.json()

    itinerary = body["itinerary"]
    assert itinerary["destination"] == "Vienna"
    assert itinerary["startDate"] == "2026-09-10"
    assert len(itinerary["days"]) == 3
    # Structure preserved: first day still has two items.
    assert len(itinerary["days"][0]["items"]) == 2
    # Deterministic rename.
    assert itinerary["days"][0]["items"][0]["name"] == "Vienna version of Prague Old Town walk"
    # Dates are shifted from the target start date.
    assert itinerary["days"][0]["date"] == "2026-09-10"
    assert itinerary["days"][1]["date"] == "2026-09-11"
    assert itinerary["days"][2]["date"] == "2026-09-12"

    summary = body["adaptationSummary"]
    assert summary["sourceDurationDays"] == 3
    assert summary["targetDurationDays"] == 3
    assert summary["changedDestination"] is True
    assert summary["fallbackUsed"] is False
    assert any("verif" in warning.lower() for warning in summary["warnings"])
    assert any("availability" in warning.lower() for warning in summary["warnings"])


def test_mock_adaptation_shorter_duration_trims_days():
    target = deepcopy(TARGET)
    target["durationDays"] = 2
    response = client.post("/adapt-template", json=_payload(target=target))
    assert response.status_code == 200
    body = response.json()
    assert len(body["itinerary"]["days"]) == 2
    assert body["adaptationSummary"]["targetDurationDays"] == 2
    assert any("trim" in change.lower() for change in body["adaptationSummary"]["majorChanges"])


def test_mock_adaptation_longer_duration_adds_placeholders():
    target = deepcopy(TARGET)
    target["durationDays"] = 5
    response = client.post("/adapt-template", json=_payload(target=target))
    assert response.status_code == 200
    body = response.json()
    days = body["itinerary"]["days"]
    assert len(days) == 5
    # The extended days are flexible exploration placeholders.
    assert "Flexible exploration" in days[4]["title"]
    assert days[4]["date"] == "2026-09-14"


def test_rejects_invalid_duration():
    target = deepcopy(TARGET)
    target["durationDays"] = 0
    response = client.post("/adapt-template", json=_payload(target=target))
    assert response.status_code == 422


def test_rejects_missing_destination():
    target = deepcopy(TARGET)
    target["destination"] = ""
    response = client.post("/adapt-template", json=_payload(target=target))
    assert response.status_code == 422


def test_returns_strict_itinerary_schema():
    response = client.post("/adapt-template", json=_payload())
    assert response.status_code == 200
    body = response.json()
    assert set(body.keys()) == {"itinerary", "adaptationSummary"}
    for day in body["itinerary"]["days"]:
        assert day["title"]
        assert len(day["items"]) >= 1
        for item in day["items"]:
            assert item["name"]
            assert item["type"] in {"place", "food", "activity", "transport", "rest"}


def test_prompt_builder_includes_preservation_and_json_only():
    request = TemplateAdaptationRequest.model_validate(_payload())
    prompt = build_template_adaptation_prompt(request)
    assert "PRESERVATION RULES" in prompt
    assert "ADAPTATION RULES" in prompt
    assert "DURATION ADAPTATION RULES" in prompt
    assert "Return ONLY valid JSON" in prompt
    assert "Do not include any text outside the JSON" in prompt
    # Safety constraints present.
    assert "do not guarantee availability" in prompt.lower()
    assert "estimates" in prompt.lower()
    # Special instructions passed through.
    assert "first-time visitors" in prompt
    # Private template metadata must not leak into the prompt.
    assert "createdFromTripId" not in prompt


def test_prompt_builder_omits_private_metadata_when_present():
    payload = _payload()
    # Even if a caller leaves stray metadata on the template, the strict schema
    # drops it so it can never reach the prompt.
    payload["template"]["metadata"] = {"createdFromTripId": "secret-trip-id"}
    request = TemplateAdaptationRequest.model_validate(payload)
    prompt = build_template_adaptation_prompt(request)
    assert "secret-trip-id" not in prompt
