# ruff: noqa: E501
"""Trip recap generation constrained to the safe source summary contract."""

from __future__ import annotations

import json
import logging
from typing import Protocol

import httpx
from pydantic import ValidationError

from app.config import Settings
from app.schemas.trip_recap import GenerateTripRecapRequest, GenerateTripRecapResponse

logger = logging.getLogger(__name__)


class TripRecapGenerator(Protocol):
    def generate(self, request: GenerateTripRecapRequest) -> GenerateTripRecapResponse: ...


class MockTripRecapGenerator:
    """Produces a deterministic recap without adding facts to the source summary."""

    def generate(self, request: GenerateTripRecapRequest) -> GenerateTripRecapResponse:
        source = request.source_summary
        trip = _section(source, "trip")
        itinerary = _section(source, "itineraryOutcome")
        budget = _section(source, "budgetOutcome")
        route = _section(source, "routeOutcome")
        checklist = _section(source, "checklistOutcome")
        verification = _section(source, "verificationOutcome")
        planned = _number(itinerary, "plannedItemCount")
        done = _number(itinerary, "doneItemCount")
        skipped = _number(itinerary, "skippedItemCount")
        delayed = _number(itinerary, "delayedItemCount")
        unknown = _number(itinerary, "unknownItemCount")
        completion = done / planned if planned else 0
        completed = _strings(itinerary, "topCompletedItems")
        skipped_items = _strings(itinerary, "topSkippedItems")
        modes = _strings(route, "transportModes")
        route_issues = _strings(route, "issues")
        verification_issues = _strings(verification, "issues")
        lessons = _lessons(skipped, _number(route, "unverifiedTransportCount"), budget)
        payload = {
            "schemaVersion": "trip_recap_v1",
            "title": f"{_text(trip, 'title', 'Trip')} Recap",
            "summary": f"{done} of {planned} planned itinerary items were marked done. This private recap is editable; review details before finalizing.",
            "highlights": [
                {"title": item, "description": "Completed itinerary moment."}
                for item in completed[:5]
            ],
            "plannedVsActual": {
                "plannedItemCount": planned,
                "doneItemCount": done,
                "skippedItemCount": skipped,
                "delayedItemCount": delayed,
                "unknownItemCount": unknown,
                "completionRate": completion,
                "notes": "Based on recorded Travel Day item statuses.",
                "skippedItems": skipped_items[:5],
                "delayedItems": [f"{delayed} itinerary item(s) were marked delayed."]
                if delayed
                else [],
            },
            "budget": {
                "plannedTotal": budget.get("plannedTotal"),
                "actualTotal": budget.get("actualTotal"),
                "varianceAmount": budget.get("variance"),
                "variancePercent": _variance_percent(budget),
                "receiptCoveragePercent": _number(budget, "receiptCoveragePercent"),
                "topCategories": budget.get("topCategories") or [],
                "notes": "Actual spend is based on tracked expenses."
                if budget.get("actualTotal")
                else "",
            },
            "routeAndTransport": {
                "summary": "Recorded transport modes: " + ", ".join(modes) + "."
                if modes
                else "No route or selected transport outcomes were recorded.",
                "issues": route_issues[:6],
                "successfulModes": modes,
                "problemModes": [],
            },
            "verification": {
                "summary": _text(verification, "summary", "No verification summary was available."),
                "issues": verification_issues[:6],
            },
            "checklistAndReminders": {
                "completedChecklistItems": _number(checklist, "completedChecklistItems"),
                "totalChecklistItems": _number(checklist, "totalChecklistItems"),
                "completedReminders": _number(checklist, "completedReminders"),
                "totalReminders": _number(checklist, "totalReminders"),
                "notes": "Completion counts are based on tracked checklist items and reminders.",
            },
            "lessonsLearned": lessons,
            "futurePreferences": _learning_candidates(source)
            if request.include_learning_candidates
            else [],
            "templateSuggestion": {
                "recommended": completion >= 0.6,
                "title": f"{_text(trip, 'destination', 'Trip')} trip template",
                "reason": "A reusable template keeps only safe itinerary structure and planning details.",
            },
            "userEditableNotes": "",
        }
        return GenerateTripRecapResponse(recap=payload, warnings=[], assumptions=[])


class OllamaTripRecapGenerator:
    def __init__(self, settings: Settings, fallback: TripRecapGenerator | None = None) -> None:
        self._settings = settings
        self._fallback = fallback or MockTripRecapGenerator()

    def generate(self, request: GenerateTripRecapRequest) -> GenerateTripRecapResponse:
        try:
            with httpx.Client(
                base_url=self._settings.ollama_base_url.rstrip("/"),
                timeout=self._settings.trip_recap_timeout_seconds,
            ) as client:
                response = client.post(
                    "/api/generate",
                    json={
                        "model": self._settings.ollama_model,
                        "prompt": _prompt(request),
                        "stream": False,
                        "format": "json",
                        "options": {
                            "temperature": min(max(self._settings.ollama_temperature, 0), 0.2),
                            "num_predict": min(max(self._settings.ollama_num_predict, 512), 1800),
                        },
                    },
                )
            response.raise_for_status()
            raw = response.json().get("response")
            if not isinstance(raw, str):
                raise ValueError("Ollama recap response is missing")
            parsed = json.loads(raw)
            return GenerateTripRecapResponse.model_validate(
                {"recap": parsed, "warnings": [], "assumptions": []}
            )
        except (httpx.HTTPError, ValueError, ValidationError, json.JSONDecodeError) as exc:
            if self._settings.trip_recap_fallback_enabled:
                logger.warning(
                    "Trip recap generation failed; using deterministic fallback",
                    extra={"errorType": type(exc).__name__},
                )
                return self._fallback.generate(request)
            raise RuntimeError("Trip recap generation is unavailable") from exc


def get_trip_recap_generator(settings: Settings) -> TripRecapGenerator:
    if settings.trip_recap_mode.strip().lower() == "ollama":
        return OllamaTripRecapGenerator(settings)
    return MockTripRecapGenerator()


def _prompt(request: GenerateTripRecapRequest) -> str:
    source = json.dumps(request.source_summary, ensure_ascii=False, separators=(",", ":"))
    return f"""
Return strict JSON only for a private editable trip recap using this exact JSON shape: schemaVersion, title, summary, highlights, plannedVsActual, budget, routeAndTransport, verification, checklistAndReminders, lessonsLearned, futurePreferences, templateSuggestion, userEditableNotes.
Rules: use only SOURCE_SUMMARY. Do not invent places, expenses, bookings, outcomes, or provider facts. Clearly distinguish planned versus recorded actual outcomes. Do not include raw receipts, OCR, calendar details, comments, secrets, passwords, tokens, or public/social copy. Learning candidates must remain approved=false. Keep keys and feedback enums in English; write user-facing text in {request.language}.
SOURCE_SUMMARY: {source}
""".strip()


def _section(source: dict, key: str) -> dict:
    value = source.get(key)
    return value if isinstance(value, dict) else {}


def _number(source: dict, key: str) -> int:
    value = source.get(key, 0)
    return int(value) if isinstance(value, (int, float)) and value >= 0 else 0


def _text(source: dict, key: str, default: str) -> str:
    value = source.get(key)
    return value.strip()[:160] if isinstance(value, str) and value.strip() else default


def _strings(source: dict, key: str) -> list[str]:
    value = source.get(key)
    return (
        [item.strip()[:240] for item in value if isinstance(item, str) and item.strip()][:12]
        if isinstance(value, list)
        else []
    )


def _variance_percent(budget: dict) -> float | None:
    planned, variance = budget.get("plannedTotal"), budget.get("variance")
    if not isinstance(planned, dict) or not isinstance(variance, dict):
        return None
    amount, diff = planned.get("amount"), variance.get("amount")
    return diff / amount * 100 if isinstance(amount, (int, float)) and amount else None


def _lessons(skipped: int, unverified_transport: int, budget: dict) -> list[str]:
    lessons: list[str] = []
    if skipped:
        lessons.append("Review skipped activities before reusing this itinerary.")
    if unverified_transport:
        lessons.append("Verify selected transport closer to departure for future trips.")
    if budget.get("actualTotal") and _number(budget, "receiptCoveragePercent") < 100:
        lessons.append("Add receipts consistently to make future budget comparisons more complete.")
    return lessons or ["Keep tracking statuses and expenses to make the next recap more useful."]


def _learning_candidates(source: dict) -> list[dict]:
    route, itinerary, budget = (
        _section(source, "routeOutcome"),
        _section(source, "itineraryOutcome"),
        _section(source, "budgetOutcome"),
    )
    candidates: list[dict] = []
    if (
        "train" in _strings(route, "transportModes")
        and _number(route, "unverifiedTransportCount") == 0
    ):
        candidates.append(
            {
                "feedbackType": "prefer_next_time",
                "label": "Prefer train routes",
                "entityType": "transport_mode",
                "value": "train",
                "metadata": {"transport": "train"},
                "approved": False,
            }
        )
    if _number(itinerary, "skippedItemCount"):
        candidates.append(
            {
                "feedbackType": "pace_too_packed",
                "label": "Leave more room for changes",
                "entityType": "general",
                "value": "pace",
                "metadata": {},
                "approved": False,
            }
        )
    if isinstance(budget.get("variance"), dict) and budget["variance"].get("amount", 0) > 0:
        candidates.append(
            {
                "feedbackType": "budget_inaccurate",
                "label": "Review budget estimates for a similar trip",
                "entityType": "general",
                "value": "budget",
                "metadata": {},
                "approved": False,
            }
        )
    return candidates[:3]
