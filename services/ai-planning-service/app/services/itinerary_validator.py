import logging
import re
from dataclasses import dataclass, field
from decimal import Decimal

from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.schemas.itinerary import TRANSPORT_MODES

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}
_VALID_ITEM_TYPES = {"place", "food", "activity", "transport", "transfer", "rest"}
_TIME_PATTERN = re.compile(r"^(?:[01]\d|2[0-3]):[0-5]\d$")
_BUDGET_OVERRUN_MULTIPLIER = Decimal("1.30")
_LONG_WALK_PATTERN = re.compile(r"\b(long|extended|lengthy|walking-heavy|full-day)\s+walk", re.I)

logger = logging.getLogger(__name__)


class ItineraryValidationError(Exception):
    def __init__(self, message: str, code: str | None = None) -> None:
        super().__init__(message)
        self.message = message
        self.code = code


@dataclass(frozen=True)
class ItineraryValidationWarning:
    code: str
    message: str


@dataclass
class ItineraryValidationResult:
    warnings: list[ItineraryValidationWarning] = field(default_factory=list)


class ItineraryValidator:
    def __init__(self, *, require_item_notes: bool = True) -> None:
        self._require_item_notes = require_item_notes

    def validate(
        self,
        request: GenerateItineraryRequest,
        itinerary: ItineraryResponse,
    ) -> ItineraryValidationResult:
        result = ItineraryValidationResult()

        if len(itinerary.days) != request.days:
            raise ItineraryValidationError(
                f"Expected {request.days} itinerary day(s), received {len(itinerary.days)}",
                code="days_count_mismatch",
            )

        expected_items_per_day = _ITEMS_PER_DAY_BY_PACE.get(request.pace, 4)
        total_estimated_cost = Decimal("0")
        estimated_cost_count = 0
        food_texts: list[str] = []
        long_walk_mentions = 0

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
            day_item_texts: list[str] = []

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
                if item.end_time is not None and not _TIME_PATTERN.match(item.end_time):
                    raise ItineraryValidationError(
                        f"Day {day.day} item {item_index} has invalid endTime {item.end_time!r}",
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
                if item.type == "transfer":
                    self._validate_transfer_item(day.day, item_index, item)

                cost_amount = (
                    item.estimated_cost.amount if item.estimated_cost is not None else None
                )
                if cost_amount is not None:
                    if cost_amount < 0:
                        raise ItineraryValidationError(
                            f"Day {day.day} item {item_index} estimated cost cannot be negative",
                            code="negative_cost",
                        )
                    total_estimated_cost += cost_amount
                    estimated_cost_count += 1

                combined_text = " ".join(
                    value for value in [item.type, item.name, item.note or ""] if value
                )
                day_item_texts.append(combined_text)
                result.warnings.extend(
                    _avoidance_warnings(request, day.day, item_index, combined_text)
                )
                if item.type == "food":
                    food_texts.append(combined_text.casefold())
                if _LONG_WALK_PATTERN.search(combined_text):
                    long_walk_mentions += 1

            result.warnings.extend(_weather_warnings(request, day.day, day_item_texts))

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

        result.warnings.extend(_dietary_warnings(request, food_texts))
        result.warnings.extend(_walking_warnings(request, long_walk_mentions))
        if result.warnings:
            logger.warning(
                "Itinerary validation completed with personalization warnings",
                extra={
                    "trip_id": str(request.trip_id),
                    "warning_codes": [warning.code for warning in result.warnings],
                },
            )
        return result

    def _validate_transfer_item(self, day_number: int, item_index: int, item) -> None:
        transfer = item.transfer
        if transfer is None:
            raise ItineraryValidationError(
                f"Day {day_number} item {item_index} transfer details are required",
                code="missing_transfer_details",
            )
        if transfer.mode not in TRANSPORT_MODES:
            raise ItineraryValidationError(
                f"Day {day_number} item {item_index} has unsupported transfer mode",
                code="invalid_transfer_mode",
            )
        if item.transport_mode is not None and item.transport_mode not in TRANSPORT_MODES:
            raise ItineraryValidationError(
                f"Day {day_number} item {item_index} has unsupported transportMode",
                code="invalid_transfer_mode",
            )
        cost = item.estimated_cost or transfer.estimated_cost
        if cost is not None and cost.category not in {None, "transport"}:
            raise ItineraryValidationError(
                f"Day {day_number} item {item_index} transfer estimatedCost must use transport category",
                code="invalid_transfer_cost_category",
            )


def _avoidance_warnings(
    request: GenerateItineraryRequest,
    day_number: int,
    item_index: int,
    combined_text: str,
) -> list[ItineraryValidationWarning]:
    preferences = request.user_preferences
    if preferences is None or not preferences.avoid:
        return []

    normalized_text = combined_text.casefold()
    warnings: list[ItineraryValidationWarning] = []
    for avoid_term in preferences.avoid:
        normalized_avoid_term = avoid_term.strip().casefold()
        if not normalized_avoid_term:
            continue
        if _term_matches(normalized_text, normalized_avoid_term):
            warnings.append(
                ItineraryValidationWarning(
                    code="avoid_term_mentioned",
                    message=(
                        f"Day {day_number} item {item_index} mentions avoided term {avoid_term!r}"
                    ),
                )
            )
    return warnings


def _term_matches(normalized_text: str, normalized_term: str) -> bool:
    terms = {normalized_term}
    if normalized_term.endswith("s") and len(normalized_term) > 1:
        terms.add(normalized_term[:-1])
    return any(term in normalized_text for term in terms)


def _dietary_warnings(
    request: GenerateItineraryRequest,
    food_texts: list[str],
) -> list[ItineraryValidationWarning]:
    preferences = request.user_preferences
    if preferences is None or not preferences.dietary_restrictions:
        return []

    restrictions = [
        restriction.strip().casefold()
        for restriction in preferences.dietary_restrictions
        if restriction.strip()
    ]
    if not restrictions:
        return []

    if not food_texts or not any(
        restriction in food_text for restriction in restrictions for food_text in food_texts
    ):
        return [
            ItineraryValidationWarning(
                code="dietary_restrictions_not_reflected",
                message="Dietary restrictions are present but food items do not mention them.",
            )
        ]
    return []


def _walking_warnings(
    request: GenerateItineraryRequest,
    long_walk_mentions: int,
) -> list[ItineraryValidationWarning]:
    preferences = request.user_preferences
    if preferences is None or preferences.max_walking_km_per_day is None:
        return []

    if preferences.max_walking_km_per_day <= 3 and long_walk_mentions >= 2:
        return [
            ItineraryValidationWarning(
                code="walking_limit_may_be_exceeded",
                message=(
                    "Low maxWalkingKmPerDay is set but itinerary repeatedly suggests long walks."
                ),
            )
        ]
    return []


def _weather_warnings(
    request: GenerateItineraryRequest,
    day_number: int,
    day_item_texts: list[str],
) -> list[ItineraryValidationWarning]:
    forecast = request.weather_forecast
    if forecast is None or day_number < 1 or day_number > len(forecast.days):
        return []

    day_weather = forecast.days[day_number - 1]
    normalized_text = " ".join(day_item_texts).casefold()
    warnings: list[ItineraryValidationWarning] = []

    if day_weather.precipitation_chance >= 60 and _looks_all_outdoor(normalized_text):
        warnings.append(
            ItineraryValidationWarning(
                code="rainy_day_lacks_indoor_backup",
                message=(
                    f"Day {day_number} has high rain chance but does not appear to include "
                    "indoor alternatives."
                ),
            )
        )

    if day_weather.temperature_max_c >= 32 and _LONG_WALK_PATTERN.search(normalized_text):
        warnings.append(
            ItineraryValidationWarning(
                code="heat_with_long_walk",
                message=f"Day {day_number} mentions a long walk during high heat.",
            )
        )

    if (day_weather.temperature_max_c <= 5 or day_weather.wind_speed_kph >= 35) and any(
        term in normalized_text for term in ["viewpoint", "exposed", "lookout", "rooftop"]
    ):
        warnings.append(
            ItineraryValidationWarning(
                code="cold_or_windy_exposed_stop",
                message=f"Day {day_number} may include exposed stops during cold or windy weather.",
            )
        )

    return warnings


def _looks_all_outdoor(normalized_text: str) -> bool:
    indoor_terms = ["museum", "gallery", "cafe", "restaurant", "covered", "indoor", "market"]
    return not any(term in normalized_text for term in indoor_terms)
