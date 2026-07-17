from __future__ import annotations

from datetime import date
from decimal import Decimal
from typing import Annotated, Literal
from uuid import UUID

from pydantic import Field, StringConstraints, field_serializer, field_validator

from app.schemas.itinerary import (
    AccommodationContext,
    APIModel,
    CurrentItinerary,
    OutputLanguage,
    TripRoute,
    UserPreferences,
    UserProfile,
    WeatherForecast,
    WorkspacePolicyConstraints,
    _serialize_decimal,
)
from app.schemas.observability import AIResponseMetadata
from app.schemas.planning_constraints import PlanningConstraints

NonEmptyString = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]

ChecklistCategory = Literal[
    "documents",
    "clothing",
    "electronics",
    "health_safety",
    "transport",
    "accommodation",
    "activities",
    "food_water",
    "money",
    "before_departure",
    "group_items",
    "camping_hiking",
    "weather",
    "other",
]
ChecklistPriority = Literal["low", "medium", "high", "critical"]
ChecklistItemType = Literal[
    "packing",
    "preparation",
    "booking_check",
    "document",
    "shared_group_item",
    "reminder",
    "safety_check",
    "other",
]
ChecklistSource = Literal["ai", "manual", "template", "regenerated", "system"]
ChecklistGenerationMode = Literal["full", "add_missing", "category"]


class ChecklistTripBudget(APIModel):
    amount: Decimal | None = Field(default=None, ge=Decimal("0"))
    currency: str = Field(default="EUR", min_length=3, max_length=3)

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            normalized = value.strip().upper()
            return normalized or "EUR"
        return value

    @field_serializer("amount", when_used="json")
    def serialize_amount(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


class ChecklistTrip(APIModel):
    id: UUID | None = None
    title: str | None = Field(default=None, max_length=200)
    destination: NonEmptyString
    start_date: date | None = Field(default=None, alias="startDate")
    duration_days: int = Field(ge=1, le=90, alias="durationDays")
    travelers: int = Field(default=1, ge=1, le=100)
    budget: ChecklistTripBudget | None = None
    interests: list[str] = Field(default_factory=list)
    pace: str = "balanced"
    trip_type: str = Field(default="single_destination", alias="tripType")

    @field_validator("interests", mode="before")
    @classmethod
    def default_interests(cls, value: object) -> object:
        if value is None:
            return []
        return value

    @field_validator("interests")
    @classmethod
    def normalize_interests(cls, value: list[str]) -> list[str]:
        return [item.strip().lower() for item in value if item.strip()]

    @field_validator("pace", "trip_type", mode="before")
    @classmethod
    def normalize_token(cls, value: object) -> object:
        if isinstance(value, str):
            return value.strip().lower().replace("-", "_") or value
        return value


class ChecklistGenerationOptions(APIModel):
    mode: ChecklistGenerationMode = "full"
    categories: list[ChecklistCategory] = Field(default_factory=list)
    preserve_checked_items: bool = Field(default=True, alias="preserveCheckedItems")
    preserve_manual_items: bool = Field(default=True, alias="preserveManualItems")
    replace_ai_items: bool = Field(default=False, alias="replaceAiItems")
    instructions: str | None = Field(default=None, max_length=1000)

    @field_validator("categories", mode="before")
    @classmethod
    def default_categories(cls, value: object) -> object:
        if value is None:
            return []
        return value

    @field_validator("instructions", mode="before")
    @classmethod
    def normalize_instructions(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class ExistingChecklistItem(APIModel):
    id: UUID | None = None
    title: str
    category: ChecklistCategory
    item_type: ChecklistItemType = Field(default="packing", alias="itemType")
    priority: ChecklistPriority = "medium"
    checked: bool = False
    source: ChecklistSource = "ai"


class ExistingChecklist(APIModel):
    id: UUID | None = None
    title: str | None = None
    items: list[ExistingChecklistItem] = Field(default_factory=list)


class GenerateChecklistRequest(APIModel):
    trip: ChecklistTrip
    itinerary: CurrentItinerary | None = None
    route: TripRoute | None = None
    weather: WeatherForecast | None = None
    accommodation: AccommodationContext | None = None
    planning_constraints: PlanningConstraints | None = Field(
        default=None, alias="planningConstraints"
    )
    group_preferences: dict[str, object] | None = Field(default=None, alias="groupPreferences")
    existing_checklist: ExistingChecklist | None = Field(default=None, alias="existingChecklist")
    generation_options: ChecklistGenerationOptions = Field(
        default_factory=ChecklistGenerationOptions,
        alias="generationOptions",
    )
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None, alias="workspacePolicyConstraints"
    )


class GeneratedChecklistItem(APIModel):
    title: NonEmptyString = Field(max_length=120)
    description: str = Field(default="", max_length=500)
    category: ChecklistCategory
    item_type: ChecklistItemType = Field(default="packing", alias="itemType")
    priority: ChecklistPriority = "medium"
    quantity: int | None = Field(default=None, ge=1, le=99)
    due_date: date | None = Field(default=None, alias="dueDate")
    reason: str = Field(default="", max_length=500)
    related_day_number: int | None = Field(default=None, ge=1, alias="relatedDayNumber")
    related_item_index: int | None = Field(default=None, ge=0, alias="relatedItemIndex")
    related_item_id: str | None = Field(default=None, max_length=100, alias="relatedItemId")
    metadata: dict[str, object] = Field(default_factory=dict)


class GeneratedChecklistResponse(APIModel):
    title: str = Field(default="Packing & preparation checklist", max_length=120)
    summary: str = Field(default="", max_length=500)
    items: list[GeneratedChecklistItem] = Field(default_factory=list, max_length=100)
    warnings: list[str] = Field(default_factory=list, max_length=12)
    metadata: AIResponseMetadata | None = None
