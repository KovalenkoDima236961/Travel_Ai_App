import re
from decimal import Decimal

from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}
_VALID_ITEM_TYPES = {"place", "food", "activity", "transport", "rest"}
_TIME_PATTERN = re.compile(r"^(?:[01]\d|2[0-3]):[0-5]\d$")
_BUDGET_OVERRUN_MULTIPLIER = Decimal("1.30")


class ItineraryValidationError(Exception):
    def __init__(self, message: str, code: str | None = None) -> None:
        super().__init__(message)
        self.message = message
        self.code = code


class ItineraryValidator:
    def __init__(self, *, require_item_notes: bool = True) -> None:
        self._require_item_notes = require_item_notes

    def validate(
        self,
        request: GenerateItineraryRequest,
        itinerary: ItineraryResponse,
    ) -> None:
        if len(itinerary.days) != request.days:
            raise ItineraryValidationError(
                f"Expected {request.days} itinerary day(s), received {len(itinerary.days)}",
                code="days_count_mismatch",
            )

        expected_items_per_day = _ITEMS_PER_DAY_BY_PACE.get(request.pace, 4)
        total_estimated_cost = Decimal("0")
        estimated_cost_count = 0

        for expected_day_number, day in enumerate(itinerary.days, start=1):
            if day.day != expected_day_number:
                raise ItineraryValidationError(
                    f"Expected day {expected_day_number}, received day {day.day}",
                    code="invalid_day_number",
                )

            if not day.title.strip():
                raise ItineraryValidationError(
                    f"Day {day.day} title cannot be empty",
                    code="empty_title",
                )

            if len(day.items) != expected_items_per_day:
                raise ItineraryValidationError(
                    f"Day {day.day} must include exactly {expected_items_per_day} item(s)",
                    code="invalid_item_count",
                )

            previous_time: str | None = None
            seen_times: set[str] = set()
            seen_item_names: set[str] = set()

            for item_index, item in enumerate(day.items, start=1):
                if item.type not in _VALID_ITEM_TYPES:
                    raise ItineraryValidationError(
                        f"Day {day.day} item {item_index} has unsupported type {item.type!r}",
                        code="invalid_item_type",
                    )

                if not _TIME_PATTERN.match(item.time):
                    raise ItineraryValidationError(
                        f"Day {day.day} item {item_index} has invalid time {item.time!r}",
                        code="invalid_time_format",
                    )

                if item.time in seen_times:
                    raise ItineraryValidationError(
                        f"Day {day.day} contains duplicate time {item.time}",
                        code="duplicate_time",
                    )
                seen_times.add(item.time)

                if previous_time is not None and item.time < previous_time:
                    raise ItineraryValidationError(
                        f"Day {day.day} items must be sorted by time ascending",
                        code="unordered_times",
                    )
                previous_time = item.time

                normalized_name = item.name.strip().casefold()
                if not normalized_name:
                    raise ItineraryValidationError(
                        f"Day {day.day} item {item_index} name cannot be empty",
                        code="empty_item_name",
                    )

                if normalized_name in seen_item_names:
                    raise ItineraryValidationError(
                        f"Day {day.day} contains duplicate item name {item.name!r}",
                        code="duplicate_item",
                    )
                seen_item_names.add(normalized_name)

                if self._require_item_notes and not (item.note or "").strip():
                    raise ItineraryValidationError(
                        f"Day {day.day} item {item_index} note cannot be empty",
                        code="empty_item_note",
                    )

                if item.estimated_cost is not None:
                    if item.estimated_cost < 0:
                        raise ItineraryValidationError(
                            f"Day {day.day} item {item_index} estimated cost cannot be negative",
                            code="negative_cost",
                        )
                    total_estimated_cost += item.estimated_cost
                    estimated_cost_count += 1

        if (
            request.budget_amount is not None
            and request.budget_amount > 0
            and estimated_cost_count > 0
            and total_estimated_cost > request.budget_amount * _BUDGET_OVERRUN_MULTIPLIER
        ):
            raise ItineraryValidationError(
                "Total estimated itinerary cost exceeds the requested budget by more than 30%",
                code="budget_exceeded",
            )
