"""Validation for AI template adaptation output.

Unlike itinerary generation, adaptation deliberately preserves the source
template's per-day activity density, so it does NOT enforce the pace-based
items-per-day rule. It validates structural correctness (day count matches the
target duration, valid item types/times, non-negative costs) and surfaces
non-blocking warnings (budget probably too low, intense compression, uncertain
prices, availability must be checked, limited destination context).
"""

import re
from dataclasses import dataclass, field
from decimal import Decimal

from app.schemas.template_adaptation import (
    TEMPLATE_ITEM_TYPES,
    AdaptedItinerary,
    TemplateAdaptationRequest,
)

_TIME_PATTERN = re.compile(r"^(?:[01]\d|2[0-3]):[0-5]\d$")
_BUDGET_LOW_MULTIPLIER = Decimal("0.6")
_HEAVY_COMPRESSION_RATIO = Decimal("0.6")


class TemplateAdaptationValidationError(Exception):
    def __init__(self, message: str, code: str | None = None) -> None:
        super().__init__(message)
        self.message = message
        self.code = code


@dataclass
class TemplateAdaptationValidationResult:
    warnings: list[str] = field(default_factory=list)


class TemplateAdaptationValidator:
    def validate(
        self,
        request: TemplateAdaptationRequest,
        itinerary: AdaptedItinerary,
    ) -> TemplateAdaptationValidationResult:
        target = request.target
        result = TemplateAdaptationValidationResult()

        if len(itinerary.days) != target.duration_days:
            raise TemplateAdaptationValidationError(
                f"Expected {target.duration_days} adapted day(s), received {len(itinerary.days)}",
                code="days_count_mismatch",
            )

        total_estimated_cost = Decimal("0")
        estimated_cost_count = 0

        for expected_day_number, day in enumerate(itinerary.days, start=1):
            if not day.title.strip():
                raise TemplateAdaptationValidationError(
                    f"Adapted day {expected_day_number} title cannot be empty",
                    code="empty_title",
                )
            if not day.items:
                raise TemplateAdaptationValidationError(
                    f"Adapted day {expected_day_number} must include at least one item",
                    code="empty_day",
                )

            for item_index, item in enumerate(day.items, start=1):
                if item.type not in TEMPLATE_ITEM_TYPES:
                    raise TemplateAdaptationValidationError(
                        f"Adapted day {expected_day_number} item {item_index} "
                        f"has unsupported type {item.type!r}",
                        code="invalid_item_type",
                    )
                if not item.name.strip():
                    raise TemplateAdaptationValidationError(
                        f"Adapted day {expected_day_number} item {item_index} name cannot be empty",
                        code="empty_item_name",
                    )
                for time_value in (item.time, item.start_time, item.end_time):
                    if time_value and not _TIME_PATTERN.match(time_value):
                        raise TemplateAdaptationValidationError(
                            f"Adapted day {expected_day_number} item {item_index} "
                            f"has invalid time {time_value!r}",
                            code="invalid_time_format",
                        )
                cost_amount = (
                    item.estimated_cost.amount if item.estimated_cost is not None else None
                )
                if cost_amount is not None:
                    if cost_amount < 0:
                        raise TemplateAdaptationValidationError(
                            f"Adapted day {expected_day_number} item {item_index} "
                            "estimated cost cannot be negative",
                            code="negative_cost",
                        )
                    total_estimated_cost += cost_amount
                    estimated_cost_count += 1

        result.warnings.extend(
            _budget_warnings(request, total_estimated_cost, estimated_cost_count)
        )
        result.warnings.extend(_duration_warnings(request))
        result.warnings.extend(_context_warnings(request))
        # Prices are always estimates and availability is never checked here.
        result.warnings.append("Estimated prices are approximate and should be verified.")
        result.warnings.append("Availability and opening hours must be checked before booking.")
        return result


def _budget_warnings(
    request: TemplateAdaptationRequest,
    total_estimated_cost: Decimal,
    estimated_cost_count: int,
) -> list[str]:
    budget = request.target.budget
    if budget is None or budget.amount <= 0 or estimated_cost_count == 0:
        return []
    if total_estimated_cost > budget.amount:
        return [
            "The requested budget may be too low for this itinerary; "
            "estimated costs exceed the target budget."
        ]
    if total_estimated_cost > budget.amount * _BUDGET_LOW_MULTIPLIER:
        return ["The requested budget leaves little room for extras; review estimated costs."]
    return []


def _duration_warnings(request: TemplateAdaptationRequest) -> list[str]:
    source = request.template.duration_days
    target = request.target.duration_days
    if source <= 0:
        return []
    if Decimal(target) < Decimal(source) * _HEAVY_COMPRESSION_RATIO:
        return [
            "The target duration is much shorter than the template; "
            "compressing days may make the plan intense."
        ]
    return []


def _context_warnings(request: TemplateAdaptationRequest) -> list[str]:
    context = request.context
    if context is None or not context.destination_context:
        return ["Destination context is limited; some suggestions may be generic."]
    return []
