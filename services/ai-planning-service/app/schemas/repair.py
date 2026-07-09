from decimal import Decimal
from typing import Any, Literal

from pydantic import Field, field_serializer, field_validator, model_validator

from app.schemas.itinerary import (
    APIModel,
    CurrencyCode,
    OutputLanguage,
    Pace,
    UserPreferences,
    UserProfile,
)

RepairMode = Literal[
    "policy_compliance",
    "reduce_budget_risk",
    "fix_schedule_risk",
    "reduce_walking",
    "add_rest_time",
    "replace_disallowed_items",
    "selected_issues",
]


def _serialize_decimal(value: Decimal | None) -> int | float | None:
    if value is None:
        return None
    if value == value.to_integral_value():
        return int(value)
    return float(value)


class RepairMoney(APIModel):
    amount: Decimal = Field(ge=Decimal("0"))
    currency: CurrencyCode = "EUR"

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper()
        return value

    @field_serializer("amount", when_used="json")
    def serialize_amount(self, value: Decimal) -> int | float:
        serialized = _serialize_decimal(value)
        return 0 if serialized is None else serialized


class RepairTripBudget(RepairMoney):
    pass


class RepairTripContext(APIModel):
    title: str | None = None
    destination: str | None = None
    start_date: str | None = Field(default=None, alias="startDate")
    duration_days: int | None = Field(default=None, ge=1, le=60, alias="durationDays")
    budget: RepairTripBudget | None = None
    travelers: int | None = Field(default=None, ge=1)
    pace: Pace | str | None = "balanced"

    @field_validator("title", "destination", "start_date", "pace", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class RepairAffected(APIModel):
    day_number: int | None = Field(default=None, ge=1, alias="dayNumber")
    item_index: int | None = Field(default=None, ge=0, alias="itemIndex")
    name: str | None = None


class RepairIssue(APIModel):
    type: str = Field(min_length=1, max_length=120)
    severity: str | None = Field(default=None, max_length=80)
    message: str = Field(default="", max_length=1000)
    affected: RepairAffected | dict[str, Any] | None = None

    @field_validator("type", "severity", "message", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            return value.strip()
        return value


class RepairConstraints(APIModel):
    repair_mode: RepairMode = Field(default="policy_compliance", alias="repairMode")
    selected_issue_types: list[str] = Field(default_factory=list, alias="selectedIssueTypes")
    preserve_confirmed_items: bool = Field(default=True, alias="preserveConfirmedItems")
    minimize_changes: bool = Field(default=True, alias="minimizeChanges")
    preserve_user_edited_items: bool = Field(default=True, alias="preserveUserEditedItems")
    do_not_change_accommodation: bool = Field(default=False, alias="doNotChangeAccommodation")
    do_not_change_dates: bool = Field(default=True, alias="doNotChangeDates")
    max_changed_items: int | None = Field(default=10, ge=1, le=50, alias="maxChangedItems")
    special_instructions: str | None = Field(
        default=None,
        max_length=1000,
        alias="specialInstructions",
    )

    @field_validator("selected_issue_types", mode="before")
    @classmethod
    def normalize_selected_issue_types(cls, value: object) -> object:
        if value is None:
            return []
        return value

    @field_validator("selected_issue_types")
    @classmethod
    def dedupe_selected_issue_types(cls, value: list[str]) -> list[str]:
        out: list[str] = []
        seen: set[str] = set()
        for item in value[:20]:
            trimmed = item.strip()
            if not trimmed:
                continue
            key = trimmed.casefold()
            if key in seen:
                continue
            seen.add(key)
            out.append(trimmed)
        return out

    @field_validator("special_instructions", mode="before")
    @classmethod
    def normalize_special_instructions(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class RepairContext(APIModel):
    user_profile: UserProfile | dict[str, Any] | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | dict[str, Any] | None = Field(
        default=None,
        alias="userPreferences",
    )
    destination_context: dict[str, Any] | None = Field(default=None, alias="destinationContext")
    weather_context: dict[str, Any] | None = Field(default=None, alias="weatherContext")


class RepairItineraryRequest(APIModel):
    itinerary: dict[str, Any]
    trip_context: RepairTripContext = Field(default_factory=RepairTripContext, alias="tripContext")
    policy: dict[str, Any] | None = None
    policy_evaluation: dict[str, Any] | None = Field(default=None, alias="policyEvaluation")
    approval_risk: dict[str, Any] | None = Field(default=None, alias="approvalRisk")
    issues: list[RepairIssue] = Field(default_factory=list, max_length=50)
    constraints: RepairConstraints = Field(default_factory=RepairConstraints)
    context: RepairContext | dict[str, Any] | None = None
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")

    @model_validator(mode="after")
    def itinerary_must_have_days(self) -> "RepairItineraryRequest":
        days = self.itinerary.get("days")
        if not isinstance(days, list) or not days:
            raise ValueError("itinerary.days must contain at least one day")
        return self


class RepairSummary(APIModel):
    repair_mode: RepairMode = Field(alias="repairMode")
    changed_item_count: int = Field(default=0, ge=0, alias="changedItemCount")
    added_item_count: int = Field(default=0, ge=0, alias="addedItemCount")
    removed_item_count: int = Field(default=0, ge=0, alias="removedItemCount")
    moved_item_count: int = Field(default=0, ge=0, alias="movedItemCount")
    estimated_cost_before: RepairMoney | None = Field(default=None, alias="estimatedCostBefore")
    estimated_cost_after: RepairMoney | None = Field(default=None, alias="estimatedCostAfter")
    major_changes: list[str] = Field(default_factory=list, alias="majorChanges")
    issues_addressed: list[str] = Field(default_factory=list, alias="issuesAddressed")
    issues_remaining: list[str] = Field(default_factory=list, alias="issuesRemaining")
    warnings: list[str] = Field(default_factory=list)


class RepairChange(APIModel):
    type: str = Field(min_length=1, max_length=80)
    day_number: int | None = Field(default=None, ge=1, alias="dayNumber")
    item_index: int | None = Field(default=None, ge=0, alias="itemIndex")
    before: dict[str, Any] | None = None
    after: dict[str, Any] | None = None
    reason: str | None = Field(default=None, max_length=500)


class RepairItineraryResponse(APIModel):
    repaired_itinerary: dict[str, Any] = Field(alias="repairedItinerary")
    repair_summary: RepairSummary = Field(alias="repairSummary")
    changes: list[RepairChange] = Field(default_factory=list)

    @model_validator(mode="after")
    def repaired_itinerary_must_have_days(self) -> "RepairItineraryResponse":
        days = self.repaired_itinerary.get("days")
        if not isinstance(days, list) or not days:
            raise ValueError("repairedItinerary.days must contain at least one day")
        return self
