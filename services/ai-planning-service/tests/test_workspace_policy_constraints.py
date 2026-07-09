from app.schemas.itinerary import GenerateItineraryRequest, RegenerateDayRequest
from app.schemas.template_adaptation import TemplateAdaptationRequest
from app.services.prompt_builder import (
    build_itinerary_prompt,
    build_regenerate_day_prompt,
    build_template_adaptation_prompt,
)

POLICY = {
    "summary": "Avoid activities after 22:00.\nPrefer public_transport and walking.",
    "rules": {"schemaVersion": 1, "rules": {}},
}


def test_generation_prompt_includes_workspace_policy() -> None:
    request = GenerateItineraryRequest.model_validate(
        {
            "tripId": "85c6c57e-e890-4bbb-bf8a-8d75ce84a536",
            "destination": "Vienna",
            "days": 1,
            "travelers": 1,
            "workspacePolicyConstraints": POLICY,
        }
    )
    prompt = build_itinerary_prompt(request)
    assert "WORKSPACE PLANNING POLICY" in prompt
    assert "Avoid activities after 22:00." in prompt
    assert "backend evaluation is authoritative" in prompt


def test_regeneration_prompt_includes_workspace_policy() -> None:
    request = RegenerateDayRequest.model_validate(
        {
            "trip": {
                "id": "85c6c57e-e890-4bbb-bf8a-8d75ce84a536",
                "destination": "Vienna",
                "days": 1,
                "travelers": 1,
            },
            "currentItinerary": {
                "days": [
                    {
                        "day": 1,
                        "title": "Day one",
                        "items": [{"time": "09:00", "type": "place", "name": "Museum"}],
                    }
                ]
            },
            "dayNumber": 1,
            "workspacePolicyConstraints": POLICY,
        }
    )
    assert "Prefer public_transport and walking." in build_regenerate_day_prompt(request)


def test_template_adaptation_prompt_includes_workspace_policy() -> None:
    request = TemplateAdaptationRequest.model_validate(
        {
            "template": {
                "schemaVersion": 1,
                "durationDays": 1,
                "days": [
                    {
                        "dayOffset": 0,
                        "title": "Day one",
                        "items": [{"name": "Museum", "type": "place"}],
                    }
                ],
            },
            "target": {
                "destination": "Vienna",
                "startDate": "2026-08-01",
                "durationDays": 1,
                "travelers": 1,
            },
            "workspacePolicyConstraints": POLICY,
        }
    )
    assert "Avoid activities after 22:00." in build_template_adaptation_prompt(request)
