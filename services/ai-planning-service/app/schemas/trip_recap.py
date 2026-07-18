# ruff: noqa: E501
"""Strict, privacy-bounded request and response models for trip recaps."""

from __future__ import annotations

import json
from typing import Any, Literal

from pydantic import Field, field_validator

from app.schemas.itinerary import APIModel, OutputLanguage

RECAP_SCHEMA_VERSION = "trip_recap_v1"
RECAP_FEEDBACK_TYPES = {
    "liked_place",
    "disliked_place",
    "too_expensive",
    "budget_worked_well",
    "budget_inaccurate",
    "too_much_walking",
    "pace_too_packed",
    "pace_too_slow",
    "route_worked_well",
    "transport_issue",
    "accommodation_issue",
    "weather_affected_plan",
    "checklist_missing_item",
    "reminder_helpful",
    "availability_issue",
    "favorite_activity_type",
    "avoid_next_time",
    "prefer_next_time",
    "other",
}
_FORBIDDEN_SOURCE_KEYS = {
    "rawtext",
    "ocr",
    "password",
    "sharetoken",
    "accesstoken",
    "calendar",
    "secret",
    "apikey",
}


class RecapMoney(APIModel):
    amount: float
    currency: str = Field(min_length=3, max_length=3)

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        return value.strip().upper() if isinstance(value, str) else value


class RecapHighlight(APIModel):
    title: str = Field(min_length=1, max_length=160)
    description: str = Field(default="", max_length=600)
    day_number: int = Field(default=0, ge=0, alias="dayNumber")
    item_id: str = Field(default="", max_length=160, alias="itemId")


class PlannedVsActual(APIModel):
    planned_item_count: int = Field(ge=0, alias="plannedItemCount")
    done_item_count: int = Field(ge=0, alias="doneItemCount")
    skipped_item_count: int = Field(ge=0, alias="skippedItemCount")
    delayed_item_count: int = Field(ge=0, alias="delayedItemCount")
    unknown_item_count: int = Field(ge=0, alias="unknownItemCount")
    completion_rate: float = Field(ge=0, le=1, alias="completionRate")
    notes: str = Field(default="", max_length=800)
    skipped_items: list[str] = Field(default_factory=list, max_length=12, alias="skippedItems")
    delayed_items: list[str] = Field(default_factory=list, max_length=12, alias="delayedItems")


class RecapCategoryTotal(APIModel):
    category: str = Field(min_length=1, max_length=80)
    total: RecapMoney


class BudgetRecap(APIModel):
    planned_total: RecapMoney | None = Field(default=None, alias="plannedTotal")
    actual_total: RecapMoney | None = Field(default=None, alias="actualTotal")
    variance_amount: RecapMoney | None = Field(default=None, alias="varianceAmount")
    variance_percent: float | None = Field(default=None, alias="variancePercent")
    receipt_coverage_percent: int = Field(default=0, ge=0, le=100, alias="receiptCoveragePercent")
    top_categories: list[RecapCategoryTotal] = Field(
        default_factory=list, max_length=8, alias="topCategories"
    )
    notes: str = Field(default="", max_length=800)


class RouteTransportRecap(APIModel):
    summary: str = Field(default="", max_length=800)
    issues: list[str] = Field(default_factory=list, max_length=12)
    successful_modes: list[str] = Field(
        default_factory=list, max_length=12, alias="successfulModes"
    )
    problem_modes: list[str] = Field(default_factory=list, max_length=12, alias="problemModes")


class VerificationRecap(APIModel):
    summary: str = Field(default="", max_length=800)
    issues: list[str] = Field(default_factory=list, max_length=12)


class ChecklistReminderRecap(APIModel):
    completed_checklist_items: int = Field(ge=0, alias="completedChecklistItems")
    total_checklist_items: int = Field(ge=0, alias="totalChecklistItems")
    completed_reminders: int = Field(ge=0, alias="completedReminders")
    total_reminders: int = Field(ge=0, alias="totalReminders")
    notes: str = Field(default="", max_length=800)


class LearningCandidate(APIModel):
    feedback_type: str = Field(alias="feedbackType")
    label: str = Field(min_length=1, max_length=240)
    entity_type: str = Field(default="", max_length=80, alias="entityType")
    entity_id: str = Field(default="", max_length=160, alias="entityId")
    value: str = Field(default="", max_length=300)
    metadata: dict[str, Any] = Field(default_factory=dict)
    approved: bool = False

    @field_validator("feedback_type")
    @classmethod
    def validate_feedback_type(cls, value: str) -> str:
        value = value.strip()
        if value not in RECAP_FEEDBACK_TYPES:
            raise ValueError("unsupported recap feedback type")
        return value


class TemplateSuggestion(APIModel):
    recommended: bool = False
    title: str = Field(default="", max_length=160)
    reason: str = Field(default="", max_length=500)


class TripRecap(APIModel):
    schema_version: Literal["trip_recap_v1"] = Field(
        default=RECAP_SCHEMA_VERSION, alias="schemaVersion"
    )
    title: str = Field(min_length=1, max_length=160)
    summary: str = Field(min_length=1, max_length=4000)
    highlights: list[RecapHighlight] = Field(default_factory=list, max_length=12)
    planned_vs_actual: PlannedVsActual = Field(alias="plannedVsActual")
    budget: BudgetRecap
    route_and_transport: RouteTransportRecap = Field(alias="routeAndTransport")
    verification: VerificationRecap
    checklist_and_reminders: ChecklistReminderRecap = Field(alias="checklistAndReminders")
    lessons_learned: list[str] = Field(default_factory=list, max_length=12, alias="lessonsLearned")
    future_preferences: list[LearningCandidate] = Field(
        default_factory=list, max_length=12, alias="futurePreferences"
    )
    template_suggestion: TemplateSuggestion = Field(
        default_factory=TemplateSuggestion, alias="templateSuggestion"
    )
    user_editable_notes: str = Field(default="", max_length=4000, alias="userEditableNotes")


class GenerateTripRecapRequest(APIModel):
    language: OutputLanguage = "en"
    source_summary: dict[str, Any] = Field(alias="sourceSummary")
    style: Literal["concise"] = "concise"
    include_learning_candidates: bool = Field(default=True, alias="includeLearningCandidates")

    @field_validator("source_summary")
    @classmethod
    def validate_safe_source_summary(cls, value: dict[str, Any]) -> dict[str, Any]:
        serialized = json.dumps(value, ensure_ascii=False, separators=(",", ":"))
        if len(serialized) > 16_000:
            raise ValueError("sourceSummary is too large")
        keys = _collect_keys(value)
        if keys & _FORBIDDEN_SOURCE_KEYS:
            raise ValueError("sourceSummary contains restricted private fields")
        return value


class GenerateTripRecapResponse(APIModel):
    recap: TripRecap
    warnings: list[str] = Field(default_factory=list, max_length=6)
    assumptions: list[str] = Field(default_factory=list, max_length=6)


def _collect_keys(value: object) -> set[str]:
    if isinstance(value, dict):
        result = {str(key).replace("_", "").lower() for key in value}
        for nested in value.values():
            result |= _collect_keys(nested)
        return result
    if isinstance(value, list):
        return set().union(*(_collect_keys(item) for item in value)) if value else set()
    return set()
