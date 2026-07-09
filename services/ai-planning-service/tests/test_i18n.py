import pytest
from pydantic import ValidationError

from app.schemas.itinerary import GenerateItineraryRequest
from app.services.itinerary_generator import MockItineraryGenerator
from app.services.prompt_builder import build_itinerary_prompt


def _request(language: str) -> GenerateItineraryRequest:
    return GenerateItineraryRequest.model_validate(
        {
            "tripId": "11111111-1111-1111-1111-111111111111",
            "destination": "Lviv",
            "days": 1,
            "travelers": 1,
            "outputLanguage": language,
        }
    )


@pytest.mark.parametrize(
    ("language", "expected"),
    [
        ("es", "Paseo matutino"),
        ("uk", "Ранкова прогулянка"),
        ("fr", "Promenade matinale"),
    ],
)
def test_mock_generation_localizes_text_but_keeps_enum_values(language: str, expected: str) -> None:
    response = MockItineraryGenerator().generate(_request(language))

    assert expected in response.days[0].items[0].name
    assert response.days[0].items[0].type == "place"
    payload = response.model_dump(by_alias=True)
    assert "days" in payload
    assert "estimatedCost" in payload["days"][0]["items"][0]


def test_prompt_includes_output_language_contract() -> None:
    prompt = build_itinerary_prompt(_request("uk"))

    assert "Write every user-facing text value in Ukrainian" in prompt
    assert "Keep all JSON keys and enum values in English" in prompt


def test_unsupported_output_language_is_rejected() -> None:
    with pytest.raises(ValidationError):
        _request("de")
