from decimal import Decimal

from app.schemas.itinerary import (
    EstimatedCost,
    GenerateItineraryRequest,
    ItineraryItem,
)
from app.services.itinerary_generator import MockItineraryGenerator


def _item(estimated_cost: object) -> ItineraryItem:
    return ItineraryItem.model_validate(
        {
            "time": "09:00",
            "type": "ticket",
            "name": "Museum",
            "note": "Visit",
            "estimatedCost": estimated_cost,
        }
    )


def test_estimated_cost_accepts_structured_object() -> None:
    item = _item(
        {
            "amount": 25.5,
            "currency": "eur",
            "category": "food",
            "confidence": "medium",
            "source": "ai",
            "note": "Lunch",
        }
    )
    cost = item.estimated_cost
    assert cost is not None
    assert cost.amount == Decimal("25.5")
    assert cost.currency == "EUR"
    assert cost.category == "food"
    assert cost.confidence == "medium"
    assert cost.source == "ai"


def test_estimated_cost_accepts_legacy_bare_number() -> None:
    item = _item(18)
    assert item.estimated_cost is not None
    assert item.estimated_cost.amount == Decimal("18")
    # Defaults applied when an amount is present.
    assert item.estimated_cost.source == "ai"
    assert item.estimated_cost.confidence == "low"
    assert item.estimated_cost.category == "other"


def test_estimated_cost_invalid_category_becomes_other() -> None:
    cost = _item({"amount": 10, "category": "mystery"}).estimated_cost
    assert cost is not None
    assert cost.category == "other"


def test_estimated_cost_invalid_currency_is_dropped() -> None:
    cost = _item({"amount": 10, "currency": "EU"}).estimated_cost
    assert cost is not None
    assert cost.currency is None


def test_estimated_cost_truncates_long_note() -> None:
    cost = EstimatedCost.model_validate({"amount": 1, "note": "x" * 400})
    assert cost.note is not None
    assert len(cost.note) == 300


def test_invalid_cost_object_repairs_to_null() -> None:
    # A non-numeric amount cannot be repaired, so the cost becomes null rather
    # than failing the whole item.
    item = _item({"amount": "abc"})
    assert item.estimated_cost is None


def test_negative_amount_is_preserved_for_downstream_rejection() -> None:
    cost = _item(-5).estimated_cost
    assert cost is not None
    assert cost.amount == Decimal("-5")


def test_mock_generator_includes_structured_costs_in_request_currency() -> None:
    request = GenerateItineraryRequest.model_validate(
        {
            "tripId": "550e8400-e29b-41d4-a716-446655440000",
            "destination": "Rome",
            "days": 2,
            "budgetAmount": 600,
            "budgetCurrency": "GBP",
            "travelers": 2,
            "interests": ["food", "history"],
            "pace": "balanced",
        }
    )
    response = MockItineraryGenerator().generate(request)

    costed = [
        item for day in response.days for item in day.items if item.estimated_cost is not None
    ]
    assert costed, "expected at least one item with a structured estimated cost"
    for item in costed:
        cost = item.estimated_cost
        assert cost.currency == "GBP"
        assert cost.source == "ai"
        assert cost.category in {
            "food",
            "transport",
            "ticket",
            "activity",
            "accommodation",
            "shopping",
            "other",
        }
