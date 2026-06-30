import json
from typing import Any

from pydantic import ValidationError

from app.schemas.itinerary import (
    ItineraryItem,
    ItineraryResponse,
    RegenerateDayResponse,
    RegenerateItemResponse,
)

_TOP_LEVEL_KEYS = {"days"}
_DAY_RESPONSE_KEYS = {"day"}
_ITEM_RESPONSE_KEYS = {"item"}
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
