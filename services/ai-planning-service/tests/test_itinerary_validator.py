from copy import deepcopy
from typing import Any

import pytest

from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.services.itinerary_validator import ItineraryValidationError, ItineraryValidator

BASE_REQUEST = {
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 1,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history"],
    "pace": "balanced",
}

ITEMS_BY_PACE = {
    "relaxed": [
        ("09:00", "place", "Historic center walk", 0),
        ("12:30", "food", "Local lunch stop", 18),
        ("15:30", "activity", "Focused museum visit", 16),
    ],
    "balanced": [
        ("09:00", "place", "Historic center walk", 0),
        ("12:30", "food", "Local lunch stop", 18),
        ("15:30", "activity", "Focused museum visit", 16),
        ("19:00", "food", "Dinner in a neighborhood district", 28),
    ],
    "intensive": [
        ("08:30", "place", "Historic center walk", 0),
        ("11:00", "activity", "Guided history route", 12),
        ("13:00", "food", "Local lunch stop", 18),
        ("15:30", "activity", "Focused museum visit", 16),
        ("20:00", "food", "Dinner in a neighborhood district", 28),
    ],
}


def _request(**overrides: Any) -> GenerateItineraryRequest:
    payload = deepcopy(BASE_REQUEST)
    payload.update(overrides)
    return GenerateItineraryRequest.model_validate(payload)


def _itinerary_body(*, days: int = 1, pace: str = "balanced") -> dict[str, Any]:
    return {
        "days": [
            {
                "day": day_number,
                "title": f"Day {day_number}: Rome practical highlights",
                "items": [
                    {
                        "time": time,
                        "type": item_type,
                        "name": name,
                        "note": f"Useful note for {name.lower()}.",
                        "estimatedCost": cost,
                    }
                    for time, item_type, name, cost in ITEMS_BY_PACE[pace]
                ],
            }
            for day_number in range(1, days + 1)
        ]
    }


def _itinerary(*, days: int = 1, pace: str = "balanced") -> ItineraryResponse:
    return ItineraryResponse.model_validate(_itinerary_body(days=days, pace=pace))


def _assert_validation_code(
    request: GenerateItineraryRequest,
    itinerary: ItineraryResponse,
    expected_code: str,
) -> None:
    with pytest.raises(ItineraryValidationError) as exc_info:
        ItineraryValidator().validate(request, itinerary)

    assert exc_info.value.code == expected_code


def test_valid_relaxed_itinerary_passes_with_three_items_per_day() -> None:
    ItineraryValidator().validate(_request(pace="relaxed"), _itinerary(pace="relaxed"))


def test_valid_balanced_itinerary_passes_with_four_items_per_day() -> None:
    ItineraryValidator().validate(_request(pace="balanced"), _itinerary(pace="balanced"))


def test_valid_intensive_itinerary_passes_with_five_items_per_day() -> None:
    ItineraryValidator().validate(_request(pace="intensive"), _itinerary(pace="intensive"))


def test_wrong_days_count_fails_with_days_count_mismatch() -> None:
    _assert_validation_code(_request(days=2), _itinerary(days=1), "days_count_mismatch")


def test_wrong_day_number_fails_with_invalid_day_number() -> None:
    body = _itinerary_body()
    body["days"][0]["day"] = 2

    _assert_validation_code(
        _request(), ItineraryResponse.model_validate(body), "invalid_day_number"
    )


def test_invalid_item_count_fails_with_invalid_item_count() -> None:
    body = _itinerary_body()
    body["days"][0]["items"].pop()

    _assert_validation_code(
        _request(), ItineraryResponse.model_validate(body), "invalid_item_count"
    )


def test_invalid_item_type_fails_with_invalid_item_type() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][0]["type"] = "museum"

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "invalid_item_type")


def test_invalid_time_format_fails_with_invalid_time_format() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][0]["time"] = "9 AM"

    _assert_validation_code(
        _request(), ItineraryResponse.model_validate(body), "invalid_time_format"
    )


def test_unordered_times_fail_with_unordered_times() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][1]["time"] = "08:00"

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "unordered_times")


def test_duplicate_times_fail_with_duplicate_time() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][1]["time"] = "09:00"

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "duplicate_time")


def test_empty_day_title_fails_with_empty_title() -> None:
    body = _itinerary_body()
    body["days"][0]["title"] = " "

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "empty_title")


def test_empty_item_name_fails_with_empty_item_name() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][0]["name"] = " "

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "empty_item_name")


def test_negative_estimated_cost_fails_with_negative_cost() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][0]["estimatedCost"] = -1

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "negative_cost")


def test_duplicate_item_name_in_same_day_fails_with_duplicate_item() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][1]["name"] = " historic center walk "

    _assert_validation_code(_request(), ItineraryResponse.model_validate(body), "duplicate_item")


def test_budget_exceeded_by_more_than_thirty_percent_fails_with_budget_exceeded() -> None:
    body = _itinerary_body()
    for item in body["days"][0]["items"]:
        item["estimatedCost"] = 40

    request = _request(budgetAmount=100)
    _assert_validation_code(
        request,
        ItineraryResponse.model_validate(body),
        "budget_exceeded",
    )


def test_budget_slightly_above_requested_amount_within_thirty_percent_passes() -> None:
    body = _itinerary_body()
    costs = [40, 40, 30, 20]
    for item, cost in zip(body["days"][0]["items"], costs, strict=True):
        item["estimatedCost"] = cost

    ItineraryValidator().validate(
        _request(budgetAmount=100), ItineraryResponse.model_validate(body)
    )


def test_avoid_term_warning_does_not_fail_validation() -> None:
    body = _itinerary_body()
    body["days"][0]["items"][3]["name"] = "Nightclub district dinner"
    request = _request(userPreferences={"avoid": ["nightclubs"]})

    result = ItineraryValidator().validate(request, ItineraryResponse.model_validate(body))

    assert [warning.code for warning in result.warnings] == ["avoid_term_mentioned"]


def test_dietary_restriction_warning_does_not_fail_validation() -> None:
    request = _request(userPreferences={"dietaryRestrictions": ["vegetarian"]})

    result = ItineraryValidator().validate(request, _itinerary())

    assert "dietary_restrictions_not_reflected" in [
        warning.code for warning in result.warnings
    ]
