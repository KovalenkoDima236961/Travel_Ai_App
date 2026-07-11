import json
from typing import Any

from pydantic import ValidationError

from app.schemas.checklist import GeneratedChecklistResponse
from app.schemas.itinerary import (
    BudgetOptimizationProposalResponse,
    ItineraryItem,
    ItineraryResponse,
    RegenerateDayResponse,
    RegenerateItemResponse,
)
from app.schemas.repair import RepairItineraryRequest, RepairItineraryResponse
from app.schemas.route_alternatives import RouteAlternativeResponse
from app.schemas.template_adaptation import TemplateAdaptationResponse

_TOP_LEVEL_KEYS = {"days"}
_DAY_RESPONSE_KEYS = {"day"}
_ITEM_RESPONSE_KEYS = {"item"}
_BUDGET_OPTIMIZATION_KEYS = {
    "summary",
    "scope",
    "dayNumber",
    "currency",
    "baseDayEstimatedTotal",
    "proposedDayEstimatedTotal",
    "estimatedSavingsAmount",
    "confidence",
    "changes",
    "preservedItems",
    "tradeoffs",
    "warnings",
    "proposedDay",
}
_REPAIR_KEYS = {"repairedItinerary", "repairSummary", "changes"}
_CHECKLIST_KEYS = {"title", "summary", "items", "warnings"}
_DAY_KEYS = {"day", "title", "items"}
_ITEM_KEYS = {"time", "type", "name", "note", "estimatedCost"}
_ITEM_TYPES = {"place", "food", "activity", "transport", "rest"}


class LLMResponseParseError(ValueError):
    """Raised when an LLM response cannot be parsed into an itinerary."""


def parse_itinerary_response(response_text: str, expected_days: int) -> ItineraryResponse:
    parsed = _parse_json(response_text)
    _ensure_exact_response_shape(parsed)

    try:
        itinerary = ItineraryResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match itinerary schema") from exc

    if len(itinerary.days) != expected_days:
        raise LLMResponseParseError(
            f"Expected {expected_days} itinerary day(s), received {len(itinerary.days)}"
        )

    for day in itinerary.days:
        if len(day.items) < 1:
            raise LLMResponseParseError(f"Day {day.day} must include at least one itinerary item")

    return itinerary


def parse_regenerate_day_response(
    response_text: str,
    expected_day_number: int,
) -> RegenerateDayResponse:
    parsed = _parse_json(response_text)
    _ensure_exact_day_response_shape(parsed)

    try:
        response = RegenerateDayResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match day replacement schema") from exc

    if response.day.day != expected_day_number:
        raise LLMResponseParseError(
            f"Expected replacement day {expected_day_number}, received {response.day.day}"
        )
    if len(response.day.items) < 1:
        raise LLMResponseParseError("Replacement day must include at least one itinerary item")
    for index, item in enumerate(response.day.items, start=1):
        _ensure_item_values_valid(item, f"Replacement day item {index}")
    if not response.day.title.strip():
        raise LLMResponseParseError("Replacement day title cannot be empty")

    return response


def parse_regenerate_item_response(response_text: str) -> RegenerateItemResponse:
    parsed = _parse_json(response_text)
    _ensure_exact_item_response_shape(parsed)

    try:
        response = RegenerateItemResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match item replacement schema") from exc

    _ensure_item_values_valid(response.item, "Replacement item")
    return response


def parse_budget_optimization_response(
    response_text: str,
    expected_day_number: int,
) -> BudgetOptimizationProposalResponse:
    parsed = _parse_json(response_text)
    _ensure_exact_budget_optimization_shape(parsed)

    try:
        response = BudgetOptimizationProposalResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError(
            "LLM response did not match budget optimization proposal schema"
        ) from exc

    if response.day_number != expected_day_number:
        raise LLMResponseParseError(
            f"Expected optimization day {expected_day_number}, received {response.day_number}"
        )
    if response.proposed_day.day != expected_day_number:
        raise LLMResponseParseError("Proposed day number does not match selected day")
    if response.estimated_savings_amount <= 0:
        raise LLMResponseParseError("Budget optimization proposal must estimate positive savings")
    if not response.changes:
        raise LLMResponseParseError("Budget optimization proposal must include changes")
    for index, item in enumerate(response.proposed_day.items, start=1):
        _ensure_item_values_valid(item, f"Proposed day item {index}")
    return response


def parse_repair_itinerary_response(
    response_text: str,
    request: RepairItineraryRequest,
) -> RepairItineraryResponse:
    parsed = _parse_json(response_text)
    _ensure_repair_response_shape(parsed)

    try:
        response = RepairItineraryResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match repair proposal schema") from exc

    original_days = request.itinerary.get("days")
    repaired_days = response.repaired_itinerary.get("days")
    if request.constraints.do_not_change_dates and isinstance(original_days, list):
        if not isinstance(repaired_days, list) or len(repaired_days) != len(original_days):
            raise LLMResponseParseError("Repair changed itinerary day count")
        original_numbers = [_day_number(day) for day in original_days]
        repaired_numbers = [_day_number(day) for day in repaired_days]
        if original_numbers != repaired_numbers:
            raise LLMResponseParseError("Repair changed itinerary day numbers")
    if not response.repair_summary.warnings:
        response.repair_summary.warnings.append(
            "Availability and prices should be checked again after repair."
        )
    return response


def parse_checklist_response(response_text: str) -> GeneratedChecklistResponse:
    parsed = _parse_json(response_text)
    _ensure_checklist_response_shape(parsed)

    try:
        response = GeneratedChecklistResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match checklist schema") from exc

    if not response.items:
        raise LLMResponseParseError("Checklist response must include at least one item")
    return response


def parse_template_adaptation_response(
    response_text: str,
    expected_days: int,
) -> TemplateAdaptationResponse:
    parsed = _parse_json(response_text)
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if "itinerary" not in parsed:
        raise LLMResponseParseError("LLM response must contain an 'itinerary' field")

    try:
        response = TemplateAdaptationResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError(
            "LLM response did not match template adaptation schema"
        ) from exc

    if len(response.itinerary.days) != expected_days:
        raise LLMResponseParseError(
            f"Expected {expected_days} adapted day(s), received {len(response.itinerary.days)}"
        )
    for day in response.itinerary.days:
        if len(day.items) < 1:
            raise LLMResponseParseError("Each adapted day must include at least one item")
    return response


def parse_route_alternatives_response(response_text: str) -> RouteAlternativeResponse:
    parsed = _parse_json(response_text)
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if "sessionTitle" not in parsed or "alternatives" not in parsed:
        raise LLMResponseParseError(
            "LLM response must contain sessionTitle and alternatives fields"
        )

    try:
        response = RouteAlternativeResponse.model_validate(parsed)
    except ValidationError as exc:
        raise LLMResponseParseError("LLM response did not match route alternatives schema") from exc

    if not response.alternatives:
        raise LLMResponseParseError("Route alternatives response must include alternatives")
    return response


def _parse_json(response_text: str) -> Any:
    direct_candidates = [
        response_text.strip(),
        _strip_markdown_code_fence(response_text.strip()),
    ]

    for candidate in direct_candidates:
        if not candidate:
            continue
        try:
            return json.loads(candidate)
        except json.JSONDecodeError:
            continue

    json_object = _extract_first_json_object(response_text)
    try:
        return json.loads(json_object)
    except json.JSONDecodeError as exc:
        raise LLMResponseParseError("LLM response did not contain valid JSON") from exc


def _strip_markdown_code_fence(text: str) -> str:
    if not text.startswith("```"):
        return text

    lines = text.splitlines()
    if lines and lines[0].strip().startswith("```"):
        lines = lines[1:]
    if lines and lines[-1].strip() == "```":
        lines = lines[:-1]
    return "\n".join(lines).strip()


def _extract_first_json_object(text: str) -> str:
    start = text.find("{")
    if start == -1:
        raise LLMResponseParseError("LLM response did not contain a JSON object")

    depth = 0
    in_string = False
    escaped = False

    for index in range(start, len(text)):
        char = text[index]

        if in_string:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
            continue

        if char == '"':
            in_string = True
        elif char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return text[start : index + 1]

    raise LLMResponseParseError("LLM response contained an incomplete JSON object")


def _ensure_exact_response_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _TOP_LEVEL_KEYS:
        raise LLMResponseParseError("LLM response must contain only the top-level 'days' field")

    days = parsed["days"]
    if not isinstance(days, list):
        raise LLMResponseParseError("LLM response 'days' field must be a list")

    for day_index, day in enumerate(days, start=1):
        if not isinstance(day, dict):
            raise LLMResponseParseError(f"Day {day_index} must be a JSON object")
        if set(day.keys()) != _DAY_KEYS:
            raise LLMResponseParseError(
                f"Day {day_index} must contain only day, title, and items fields"
            )
        if not isinstance(day["items"], list):
            raise LLMResponseParseError(f"Day {day_index} items field must be a list")

        for item_index, item in enumerate(day["items"], start=1):
            if not isinstance(item, dict):
                raise LLMResponseParseError(
                    f"Day {day_index} item {item_index} must be a JSON object"
                )
            if set(item.keys()) != _ITEM_KEYS:
                raise LLMResponseParseError(
                    f"Day {day_index} item {item_index} must contain exactly "
                    "the itinerary item fields"
                )
            if not isinstance(item["type"], str) or item["type"] not in _ITEM_TYPES:
                raise LLMResponseParseError(
                    f"Day {day_index} item {item_index} has unsupported type"
                )


def _ensure_exact_day_response_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _DAY_RESPONSE_KEYS:
        raise LLMResponseParseError("LLM response must contain only the top-level 'day' field")

    day = parsed["day"]
    if not isinstance(day, dict):
        raise LLMResponseParseError("Replacement day must be a JSON object")
    if set(day.keys()) != _DAY_KEYS:
        raise LLMResponseParseError("Replacement day must contain only day, title, and items")
    if not isinstance(day["items"], list):
        raise LLMResponseParseError("Replacement day items field must be a list")

    for item_index, item in enumerate(day["items"], start=1):
        _ensure_exact_item_shape(item, f"Replacement day item {item_index}")


def _ensure_exact_item_response_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _ITEM_RESPONSE_KEYS:
        raise LLMResponseParseError("LLM response must contain only the top-level 'item' field")
    _ensure_exact_item_shape(parsed["item"], "Replacement item")


def _ensure_exact_budget_optimization_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _BUDGET_OPTIMIZATION_KEYS:
        raise LLMResponseParseError("LLM response must contain exactly the proposal fields")
    proposed_day = parsed["proposedDay"]
    if not isinstance(proposed_day, dict):
        raise LLMResponseParseError("proposedDay must be a JSON object")
    if set(proposed_day.keys()) != _DAY_KEYS:
        raise LLMResponseParseError("proposedDay must contain only day, title, and items")
    if not isinstance(proposed_day["items"], list):
        raise LLMResponseParseError("proposedDay.items must be a list")
    for item_index, item in enumerate(proposed_day["items"], start=1):
        _ensure_exact_item_shape(item, f"Proposed day item {item_index}")


def _ensure_repair_response_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _REPAIR_KEYS:
        raise LLMResponseParseError("LLM repair response must contain exact proposal fields")
    repaired = parsed["repairedItinerary"]
    summary = parsed["repairSummary"]
    changes = parsed["changes"]
    if not isinstance(repaired, dict):
        raise LLMResponseParseError("repairedItinerary must be a JSON object")
    days = repaired.get("days")
    if not isinstance(days, list) or not days:
        raise LLMResponseParseError("repairedItinerary.days must be a non-empty list")
    if not isinstance(summary, dict):
        raise LLMResponseParseError("repairSummary must be a JSON object")
    if not isinstance(changes, list):
        raise LLMResponseParseError("changes must be a list")
    for day_index, day in enumerate(days, start=1):
        if not isinstance(day, dict):
            raise LLMResponseParseError(f"Repair day {day_index} must be a JSON object")
        items = day.get("items")
        if not isinstance(items, list) or not items:
            raise LLMResponseParseError(f"Repair day {day_index} items must be non-empty")
        for item_index, item in enumerate(items, start=1):
            if not isinstance(item, dict):
                raise LLMResponseParseError(
                    f"Repair day {day_index} item {item_index} must be a JSON object"
                )
            for field in ("time", "type", "name"):
                if not isinstance(item.get(field), str) or not item[field].strip():
                    raise LLMResponseParseError(
                        f"Repair day {day_index} item {item_index} field {field} is required"
                    )


def _ensure_checklist_response_shape(parsed: Any) -> None:
    if not isinstance(parsed, dict):
        raise LLMResponseParseError("LLM response must be a JSON object")
    if set(parsed.keys()) != _CHECKLIST_KEYS:
        raise LLMResponseParseError(
            "LLM checklist response must contain exactly title, summary, items, and warnings"
        )
    if not isinstance(parsed["items"], list) or not parsed["items"]:
        raise LLMResponseParseError("Checklist items must be a non-empty list")
    if not isinstance(parsed["warnings"], list):
        raise LLMResponseParseError("Checklist warnings must be a list")
    for index, item in enumerate(parsed["items"], start=1):
        if not isinstance(item, dict):
            raise LLMResponseParseError(f"Checklist item {index} must be a JSON object")
        for field in ("title", "category", "itemType", "priority"):
            if not str(item.get(field) or "").strip():
                raise LLMResponseParseError(f"Checklist item {index} missing {field}")


def _day_number(value: Any) -> int | None:
    if isinstance(value, dict):
        day = value.get("day")
        return day if isinstance(day, int) else None
    return None


def _ensure_exact_item_shape(item: Any, label: str) -> None:
    if not isinstance(item, dict):
        raise LLMResponseParseError(f"{label} must be a JSON object")
    if set(item.keys()) != _ITEM_KEYS:
        raise LLMResponseParseError(f"{label} must contain exactly the itinerary item fields")
    if not isinstance(item["type"], str) or item["type"] not in _ITEM_TYPES:
        raise LLMResponseParseError(f"{label} has unsupported type")


def _ensure_item_values_valid(item: ItineraryItem, label: str) -> None:
    if not item.time.strip():
        raise LLMResponseParseError(f"{label} time cannot be empty")
    if not item.type.strip():
        raise LLMResponseParseError(f"{label} type cannot be empty")
    if not item.name.strip():
        raise LLMResponseParseError(f"{label} name cannot be empty")
    if (
        item.estimated_cost is not None
        and item.estimated_cost.amount is not None
        and item.estimated_cost.amount < 0
    ):
        raise LLMResponseParseError(f"{label} estimatedCost cannot be negative")
