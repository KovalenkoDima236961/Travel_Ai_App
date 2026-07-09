"""Schemas for AI template adaptation.

Template adaptation takes a reusable, sanitized trip template and re-targets it
to a new destination, duration, budget, pace, travelers, and interests while
preserving the template's planning structure and rhythm. The request template
mirrors the sanitized ``template_json`` produced by Trip Service (schema version
1) minus any private metadata, and the response returns a reviewable, editable
itinerary plus an adaptation summary.
"""

from datetime import date
from decimal import Decimal

from pydantic import Field, field_validator

from app.schemas.itinerary import (
    APIModel,
    CurrencyCode,
    EstimatedCost,
    NonEmptyString,
    OutputLanguage,
    Pace,
    WorkspacePolicyConstraints,
    _normalize_string_list,
)

# Item types accepted in template days / adapted days. Kept in sync with the
# itinerary validator vocabulary so adapted output can persist unchanged.
TEMPLATE_ITEM_TYPES = {"place", "food", "activity", "transport", "rest"}

_MAX_INTERESTS = 20
_MAX_AVOID = 20
_MAX_INTEREST_LENGTH = 40
_MAX_AVOID_LENGTH = 80
_MAX_SPECIAL_INSTRUCTIONS = 1000


class TemplatePlaceInput(APIModel):
    """Optional place hint carried on a template item.

    Only descriptive fields are used for adaptation; provider identifiers are
    accepted for round-tripping but never influence the generated place names.
    """

    name: str | None = None
    category: str | None = None
    address: str | None = None

    @field_validator("name", "category", "address", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class TemplateItemInput(APIModel):
    """A single sanitized template item used as adaptation input."""

    name: NonEmptyString
    type: str = "activity"
    description: str | None = None
    time: str | None = None
    start_time: str | None = Field(default=None, alias="startTime")
    end_time: str | None = Field(default=None, alias="endTime")
    place: TemplatePlaceInput | None = None
    estimated_cost: EstimatedCost | None = Field(default=None, alias="estimatedCost")
    notes: str | None = None

    @field_validator("type", mode="before")
    @classmethod
    def normalize_type(cls, value: object) -> object:
        if value is None:
            return "activity"
        if isinstance(value, str):
            normalized = value.strip().lower()
            return normalized if normalized in TEMPLATE_ITEM_TYPES else "activity"
        return value

    @field_validator("description", "time", "start_time", "end_time", "notes", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class TemplateDayInput(APIModel):
    """A single template day (``dayOffset`` is zero-based)."""

    day_offset: int = Field(default=0, ge=0, alias="dayOffset")
    title: str = ""
    items: list[TemplateItemInput] = Field(default_factory=list)


class TemplatePayload(APIModel):
    """Sanitized template structure (schema version 1) used for adaptation.

    Private metadata (source trip IDs, summary) is intentionally omitted from
    this schema so it never reaches the model prompt.
    """

    schema_version: int = Field(default=1, alias="schemaVersion")
    duration_days: int = Field(ge=1, le=30, alias="durationDays")
    days: list[TemplateDayInput] = Field(min_length=1)


class TemplateAdaptationBudget(APIModel):
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


class TemplateAdaptationTarget(APIModel):
    """Where and how the adapted trip should land."""

    destination: NonEmptyString
    start_date: date = Field(alias="startDate")
    duration_days: int = Field(ge=1, le=30, alias="durationDays")
    budget: TemplateAdaptationBudget | None = None
    travelers: int = Field(default=1, ge=1, le=50)
    pace: Pace = "balanced"
    interests: list[str] = Field(default_factory=list)
    avoid: list[str] = Field(default_factory=list)

    @field_validator("pace", mode="before")
    @classmethod
    def default_pace(cls, value: object) -> object:
        if value is None or (isinstance(value, str) and value.strip() == ""):
            return "balanced"
        if isinstance(value, str):
            return value.strip().lower()
        return value

    @field_validator("interests", "avoid", mode="before")
    @classmethod
    def normalize_lists(cls, value: object) -> object:
        return _normalize_string_list(value)

    @field_validator("interests")
    @classmethod
    def limit_interests(cls, value: list[str]) -> list[str]:
        trimmed = [item[:_MAX_INTEREST_LENGTH] for item in value][:_MAX_INTERESTS]
        return trimmed

    @field_validator("avoid")
    @classmethod
    def limit_avoid(cls, value: list[str]) -> list[str]:
        return [item[:_MAX_AVOID_LENGTH] for item in value][:_MAX_AVOID]


class TemplateAdaptationConstraints(APIModel):
    preserve_structure: bool = Field(default=True, alias="preserveStructure")
    adapt_costs: bool = Field(default=True, alias="adaptCosts")
    preserve_meal_structure: bool = Field(default=True, alias="preserveMealStructure")
    preserve_activity_density: bool = Field(default=True, alias="preserveActivityDensity")
    special_instructions: str | None = Field(
        default=None,
        max_length=_MAX_SPECIAL_INSTRUCTIONS,
        alias="specialInstructions",
    )

    @field_validator("special_instructions", mode="before")
    @classmethod
    def normalize_special_instructions(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class AdaptationContext(APIModel):
    """Optional non-sensitive context passed through for prompt personalization."""

    user_profile: dict | None = Field(default=None, alias="userProfile")
    user_preferences: dict | None = Field(default=None, alias="userPreferences")
    destination_context: dict | None = Field(default=None, alias="destinationContext")
    weather_context: dict | None = Field(default=None, alias="weatherContext")


class TemplateAdaptationRequest(APIModel):
    template: TemplatePayload
    target: TemplateAdaptationTarget
    constraints: TemplateAdaptationConstraints = Field(
        default_factory=TemplateAdaptationConstraints
    )
    context: AdaptationContext | None = None
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None, alias="workspacePolicyConstraints"
    )


class AdaptedPlace(APIModel):
    name: str | None = None
    category: str | None = None


class AdaptedItem(APIModel):
    name: NonEmptyString
    type: str = "activity"
    description: str | None = None
    time: str | None = None
    start_time: str | None = Field(default=None, alias="startTime")
    end_time: str | None = Field(default=None, alias="endTime")
    place: AdaptedPlace | None = None
    estimated_cost: EstimatedCost | None = Field(default=None, alias="estimatedCost")
    notes: str | None = None

    @field_validator("type", mode="before")
    @classmethod
    def normalize_type(cls, value: object) -> object:
        if value is None:
            return "activity"
        if isinstance(value, str):
            normalized = value.strip().lower()
            return normalized if normalized in TEMPLATE_ITEM_TYPES else "activity"
        return value


class AdaptedDay(APIModel):
    day_date: date | None = Field(default=None, alias="date")
    title: NonEmptyString
    items: list[AdaptedItem] = Field(min_length=1)


class AdaptedItinerary(APIModel):
    title: NonEmptyString
    destination: NonEmptyString
    start_date: date = Field(alias="startDate")
    days: list[AdaptedDay] = Field(min_length=1)


class AdaptationSummary(APIModel):
    source_duration_days: int = Field(ge=1, alias="sourceDurationDays")
    target_duration_days: int = Field(ge=1, alias="targetDurationDays")
    preserved_structure: bool = Field(default=True, alias="preservedStructure")
    changed_destination: bool = Field(default=True, alias="changedDestination")
    fallback_used: bool = Field(default=False, alias="fallbackUsed")
    major_changes: list[str] = Field(default_factory=list, alias="majorChanges")
    warnings: list[str] = Field(default_factory=list)


class TemplateAdaptationResponse(APIModel):
    itinerary: AdaptedItinerary
    adaptation_summary: AdaptationSummary = Field(alias="adaptationSummary")
