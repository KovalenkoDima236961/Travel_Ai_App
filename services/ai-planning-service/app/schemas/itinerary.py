import re
from datetime import date
from decimal import Decimal, InvalidOperation
from typing import Annotated, Literal
from uuid import UUID

from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    StringConstraints,
    field_serializer,
    field_validator,
    model_validator,
)

NonEmptyString = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]
CurrencyCode = Annotated[str, StringConstraints(strip_whitespace=True, min_length=3, max_length=3)]
Pace = Literal["relaxed", "balanced", "intensive"]
OutputLanguage = Literal["en", "es", "uk", "fr"]

# Item-level cost-estimate vocabularies, kept in sync with Trip Service and the
# web client.
COST_CATEGORIES = {
    "food",
    "transport",
    "ticket",
    "activity",
    "accommodation",
    "shopping",
    "other",
}
COST_CONFIDENCES = {"low", "medium", "high"}
COST_SOURCES = {"ai", "manual", "provider"}
ACCOMMODATION_TYPES = {"hotel", "hostel", "apartment", "guesthouse", "home", "other"}
_CURRENCY_PATTERN = re.compile(r"^[A-Z]{3}$")
_MAX_COST_NOTE = 300


def _serialize_decimal(value: Decimal | None) -> int | float | None:
    if value is None:
        return None
    if value == value.to_integral_value():
        return int(value)
    return float(value)


class APIModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True)


class WorkspacePolicyConstraints(APIModel):
    """Trusted workspace planning guidance supplied by Trip Service.

    The AI uses this context as guidance; Trip Service evaluates the persisted
    itinerary authoritatively after generation.
    """

    summary: str = Field(max_length=5000)
    rules: dict[str, object] = Field(default_factory=dict)


def _normalize_string_list(value: object) -> object:
    if value is None:
        return []
    if not isinstance(value, list):
        return value

    normalized: list[str] = []
    seen: set[str] = set()
    for item in value:
        if not isinstance(item, str):
            continue
        trimmed = item.strip()
        if not trimmed:
            continue
        key = trimmed.casefold()
        if key in seen:
            continue
        seen.add(key)
        normalized.append(trimmed)
    return normalized


class UserProfile(APIModel):
    user_id: UUID | None = Field(default=None, alias="userId")
    display_name: str | None = Field(default=None, alias="displayName")
    home_city: str | None = Field(default=None, alias="homeCity")
    home_country: str | None = Field(default=None, alias="homeCountry")
    preferred_currency: str | None = Field(default="EUR", alias="preferredCurrency")
    preferred_language: str | None = Field(default="en", alias="preferredLanguage")

    @field_validator(
        "display_name",
        "home_city",
        "home_country",
        "preferred_currency",
        "preferred_language",
        mode="before",
    )
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value

    @field_validator("preferred_currency")
    @classmethod
    def normalize_preferred_currency(cls, value: str | None) -> str | None:
        if value is None:
            return None
        return value.upper()

    @field_validator("preferred_language")
    @classmethod
    def normalize_preferred_language(cls, value: str | None) -> str | None:
        if value is None:
            return None
        return value.lower()


class UserPreferences(APIModel):
    user_id: UUID | None = Field(default=None, alias="userId")
    travel_styles: list[str] = Field(default_factory=list, alias="travelStyles")
    pace: str | None = None
    max_walking_km_per_day: float | None = Field(default=None, alias="maxWalkingKmPerDay")
    food_preferences: list[str] = Field(default_factory=list, alias="foodPreferences")
    avoid: list[str] = Field(default_factory=list)
    preferred_transport: list[str] = Field(default_factory=list, alias="preferredTransport")
    accommodation_style: list[str] = Field(default_factory=list, alias="accommodationStyle")
    dietary_restrictions: list[str] = Field(default_factory=list, alias="dietaryRestrictions")

    @field_validator(
        "travel_styles",
        "food_preferences",
        "avoid",
        "preferred_transport",
        "accommodation_style",
        "dietary_restrictions",
        mode="before",
    )
    @classmethod
    def normalize_lists(cls, value: object) -> object:
        return _normalize_string_list(value)

    @field_validator("pace", mode="before")
    @classmethod
    def normalize_pace(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip().lower()
            return trimmed or None
        return value


class WeatherDay(APIModel):
    date: str
    condition: str
    temperature_min_c: float = Field(alias="temperatureMinC")
    temperature_max_c: float = Field(alias="temperatureMaxC")
    precipitation_chance: int = Field(ge=0, le=100, alias="precipitationChance")
    wind_speed_kph: float = Field(alias="windSpeedKph")
    summary: str
    warnings: list[str] = Field(default_factory=list)


class WeatherForecast(APIModel):
    destination: str
    provider: str | None = None
    days: list[WeatherDay] = Field(default_factory=list)


class AccommodationPlace(APIModel):
    provider: str | None = None
    provider_place_id: str | None = Field(default=None, alias="providerPlaceId")
    name: str | None = None
    address: str | None = None
    latitude: float | None = None
    longitude: float | None = None
    map_url: str | None = Field(default=None, alias="mapUrl")
    category: str | None = None
    website: str | None = None

    @field_validator(
        "provider",
        "provider_place_id",
        "name",
        "address",
        "map_url",
        "category",
        "website",
        mode="before",
    )
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class AccommodationContext(APIModel):
    name: NonEmptyString
    type: str = "other"
    address: str | None = None
    place: AccommodationPlace | None = None
    check_in_date: date | None = Field(default=None, alias="checkInDate")
    check_out_date: date | None = Field(default=None, alias="checkOutDate")
    estimated_cost: dict[str, object] | None = Field(default=None, alias="estimatedCost")
    notes: str | None = None

    @field_validator("type", mode="before")
    @classmethod
    def normalize_type(cls, value: object) -> object:
        if value is None:
            return "other"
        if isinstance(value, str):
            normalized = value.strip().lower()
            return normalized if normalized in ACCOMMODATION_TYPES else "other"
        return value

    @field_validator("address", "notes", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value

    @model_validator(mode="after")
    def check_out_after_check_in(self) -> "AccommodationContext":
        if (
            self.check_in_date is not None
            and self.check_out_date is not None
            and self.check_out_date <= self.check_in_date
        ):
            raise ValueError("checkOutDate must be after checkInDate")
        return self


class GenerateItineraryRequest(APIModel):
    trip_id: UUID = Field(alias="tripId")
    destination: NonEmptyString
    start_date: date | None = Field(default=None, alias="startDate")
    days: int = Field(ge=1, le=30)
    budget_amount: Decimal | None = Field(default=None, ge=Decimal("0"), alias="budgetAmount")
    budget_currency: CurrencyCode = Field(default="EUR", alias="budgetCurrency")
    travelers: int = Field(ge=1)
    interests: list[str] = Field(default_factory=list)
    pace: Pace = "balanced"
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    weather_forecast: WeatherForecast | None = Field(default=None, alias="weatherForecast")
    accommodation: AccommodationContext | None = None
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None, alias="workspacePolicyConstraints"
    )

    @field_validator("budget_currency", mode="before")
    @classmethod
    def default_budget_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str) and value.strip() == "":
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper()
        return value

    @field_validator("pace", mode="before")
    @classmethod
    def default_pace(cls, value: object) -> object:
        if value is None:
            return "balanced"
        if isinstance(value, str) and value.strip() == "":
            return "balanced"
        if isinstance(value, str):
            return value.strip().lower()
        return value

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


class OpeningHoursInterval(APIModel):
    day_of_week: int = Field(ge=1, le=7, alias="dayOfWeek")
    open: str
    close: str


class PlaceRef(APIModel):
    provider: str
    provider_place_id: str = Field(alias="providerPlaceId")
    name: str
    address: str
    latitude: float | None = None
    longitude: float | None = None
    rating: float | None = None
    rating_count: int | None = Field(default=None, alias="ratingCount")
    map_url: str | None = Field(default=None, alias="mapUrl")
    category: str | None = None
    website: str | None = None
    opening_hours: list[OpeningHoursInterval] = Field(default_factory=list, alias="openingHours")


class EstimatedCost(APIModel):
    """Structured item-level cost estimate.

    Soft problems are repaired during validation (unknown category -> "other",
    unknown confidence dropped, invalid currency dropped, note truncated) so a
    single bad field never fails generation. A negative amount is intentionally
    left intact so downstream validation can reject it.
    """

    amount: Decimal | None = None
    currency: str | None = None
    category: str | None = None
    confidence: str | None = None
    source: str | None = None
    note: str | None = None

    @field_validator("amount", mode="before")
    @classmethod
    def _empty_amount_to_none(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str) and value.strip() == "":
            return None
        return value

    @field_validator("currency")
    @classmethod
    def _normalize_currency(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip().upper()
        if not normalized or not _CURRENCY_PATTERN.match(normalized):
            return None
        return normalized

    @field_validator("category")
    @classmethod
    def _normalize_category(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip().lower()
        return normalized if normalized in COST_CATEGORIES else "other"

    @field_validator("confidence")
    @classmethod
    def _normalize_confidence(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip().lower()
        return normalized if normalized in COST_CONFIDENCES else None

    @field_validator("source")
    @classmethod
    def _normalize_source(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip().lower()
        return normalized if normalized in COST_SOURCES else None

    @field_validator("note")
    @classmethod
    def _normalize_note(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip()
        if not normalized:
            return None
        return normalized[:_MAX_COST_NOTE]

    @model_validator(mode="after")
    def _apply_defaults(self) -> "EstimatedCost":
        # Generated output defaults to source "ai"; a present amount gets a
        # low-confidence/other-category default when unspecified.
        if self.amount is not None:
            if self.source is None:
                self.source = "ai"
            if self.confidence is None:
                self.confidence = "low"
            if self.category is None:
                self.category = "other"
        return self

    @field_serializer("amount", when_used="json")
    def serialize_amount(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


class ItineraryItem(APIModel):
    time: str
    type: str
    name: str
    note: str | None = None
    estimated_cost: EstimatedCost | None = Field(default=None, alias="estimatedCost")

    @field_validator("estimated_cost", mode="before")
    @classmethod
    def _coerce_estimated_cost(cls, value: object) -> object:
        """Accept the structured object or the legacy bare-number form.

        An invalid/unrepairable cost object is dropped to null rather than
        failing the whole itinerary.
        """
        if value is None or isinstance(value, EstimatedCost):
            return value
        if isinstance(value, (int, float, Decimal)):
            payload: object = {"amount": value}
        elif isinstance(value, str):
            stripped = value.strip()
            if not stripped:
                return None
            try:
                payload = {"amount": Decimal(stripped)}
            except (InvalidOperation, ValueError):
                return None
        elif isinstance(value, dict):
            payload = value
        else:
            return None
        try:
            return EstimatedCost.model_validate(payload)
        except ValueError:
            return None


class ItineraryDay(APIModel):
    day: int = Field(ge=1)
    title: str
    items: list[ItineraryItem] = Field(min_length=1)


class ItineraryResponse(APIModel):
    days: list[ItineraryDay]


class PartialTrip(APIModel):
    id: UUID
    destination: NonEmptyString
    start_date: date | None = Field(default=None, alias="startDate")
    days: int = Field(ge=1, le=30)
    budget_amount: Decimal | None = Field(default=None, ge=Decimal("0"), alias="budgetAmount")
    budget_currency: CurrencyCode = Field(default="EUR", alias="budgetCurrency")
    travelers: int = Field(ge=1)
    interests: list[str] = Field(default_factory=list)
    pace: str = "balanced"

    @field_validator("budget_currency", mode="before")
    @classmethod
    def default_budget_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str) and value.strip() == "":
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper()
        return value

    @field_validator("pace", mode="before")
    @classmethod
    def default_pace(cls, value: object) -> object:
        if value is None:
            return "balanced"
        if isinstance(value, str) and value.strip() == "":
            return "balanced"
        if isinstance(value, str):
            return value.strip().lower()
        return value

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


class CurrentItineraryItem(ItineraryItem):
    place: PlaceRef | None = None


class CurrentItineraryDay(APIModel):
    day: int = Field(ge=1)
    title: str
    items: list[CurrentItineraryItem] = Field(min_length=1)


class CurrentItinerary(APIModel):
    days: list[CurrentItineraryDay] = Field(min_length=1)


class RegenerateDayRequest(APIModel):
    trip: PartialTrip
    current_itinerary: CurrentItinerary = Field(alias="currentItinerary")
    day_number: int = Field(ge=1, alias="dayNumber")
    instruction: str | None = Field(default=None, max_length=500)
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    weather_forecast: WeatherForecast | None = Field(default=None, alias="weatherForecast")
    accommodation: AccommodationContext | None = None
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None, alias="workspacePolicyConstraints"
    )

    @field_validator("instruction", mode="before")
    @classmethod
    def normalize_instruction(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value

    @model_validator(mode="after")
    def selected_day_must_exist(self) -> "RegenerateDayRequest":
        if self.selected_day() is None:
            raise ValueError("selected dayNumber does not exist in currentItinerary")
        return self

    def selected_day(self) -> CurrentItineraryDay | None:
        for day in self.current_itinerary.days:
            if day.day == self.day_number:
                return day
        return None


class RegenerateItemRequest(RegenerateDayRequest):
    item_index: int = Field(ge=0, alias="itemIndex")

    @model_validator(mode="after")
    def selected_item_must_exist(self) -> "RegenerateItemRequest":
        selected_day = self.selected_day()
        if selected_day is None:
            raise ValueError("selected dayNumber does not exist in currentItinerary")
        if self.item_index >= len(selected_day.items):
            raise ValueError("selected itemIndex does not exist in currentItinerary day")
        return self

    def selected_item(self) -> CurrentItineraryItem | None:
        selected_day = self.selected_day()
        if selected_day is None or self.item_index >= len(selected_day.items):
            return None
        return selected_day.items[self.item_index]


class RegenerateDayResponse(APIModel):
    day: ItineraryDay


class RegenerateItemResponse(APIModel):
    item: ItineraryItem


class BudgetOptimizationConstraints(APIModel):
    preserve_must_see_items: bool = Field(default=True, alias="preserveMustSeeItems")
    max_walking_increase_km: float | None = Field(default=None, ge=0, alias="maxWalkingIncreaseKm")
    keep_meal_count: bool = Field(default=True, alias="keepMealCount")
    avoid_replacing_manual_costs: bool = Field(default=True, alias="avoidReplacingManualCosts")


class BudgetOptimizationExpensiveItem(APIModel):
    item_index: int = Field(ge=0, alias="itemIndex")
    item_name: str = Field(alias="itemName")
    item_type: str | None = Field(default=None, alias="itemType")
    amount: Decimal = Field(ge=Decimal("0"))
    currency: CurrencyCode = "EUR"
    category: str | None = None
    source: str | None = None
    confidence: str | None = None
    share_of_day_total: float | None = Field(default=None, ge=0, alias="shareOfDayTotal")

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


class BudgetOptimizationContext(APIModel):
    currency: CurrencyCode = "EUR"
    trip_budget: Decimal | None = Field(default=None, ge=Decimal("0"), alias="tripBudget")
    trip_estimated_total: Decimal = Field(
        default=Decimal("0"), ge=Decimal("0"), alias="tripEstimatedTotal"
    )
    day_estimated_total: Decimal = Field(
        default=Decimal("0"), ge=Decimal("0"), alias="dayEstimatedTotal"
    )
    daily_budget_share: Decimal | None = Field(
        default=None, ge=Decimal("0"), alias="dailyBudgetShare"
    )
    target_reduction_amount: Decimal = Field(
        default=Decimal("0"), ge=Decimal("0"), alias="targetReductionAmount"
    )
    expensive_items: list[BudgetOptimizationExpensiveItem] = Field(
        default_factory=list, alias="expensiveItems"
    )

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper()
        return value

    @field_serializer(
        "trip_budget",
        "trip_estimated_total",
        "day_estimated_total",
        "daily_budget_share",
        "target_reduction_amount",
        when_used="json",
    )
    def serialize_decimal(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


class OptimizeBudgetDayRequest(APIModel):
    trip: PartialTrip
    current_itinerary: CurrentItinerary = Field(alias="currentItinerary")
    day_number: int = Field(ge=1, alias="dayNumber")
    current_day: CurrentItineraryDay = Field(alias="currentDay")
    budget_context: BudgetOptimizationContext = Field(alias="budgetContext")
    constraints: BudgetOptimizationConstraints = Field(
        default_factory=BudgetOptimizationConstraints
    )
    instruction: str | None = Field(default=None, max_length=2000)
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    weather_forecast: WeatherForecast | None = Field(default=None, alias="weatherForecast")
    accommodation: AccommodationContext | None = None
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None, alias="workspacePolicyConstraints"
    )

    @field_validator("instruction", mode="before")
    @classmethod
    def normalize_instruction(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value

    @model_validator(mode="after")
    def selected_day_must_match(self) -> "OptimizeBudgetDayRequest":
        if self.current_day.day != self.day_number:
            raise ValueError("currentDay.day must match dayNumber")
        if self.selected_day() is None:
            raise ValueError("selected dayNumber does not exist in currentItinerary")
        return self

    def selected_day(self) -> CurrentItineraryDay | None:
        for day in self.current_itinerary.days:
            if day.day == self.day_number:
                return day
        return None


class BudgetOptimizationChange(APIModel):
    type: Literal[
        "replace_item",
        "remove_item",
        "add_item",
        "modify_item_cost",
        "reorder_item",
        "keep_item",
    ]
    old_item_index: int | None = Field(default=None, ge=0, alias="oldItemIndex")
    old_item_name: str | None = Field(default=None, alias="oldItemName")
    new_item_index: int | None = Field(default=None, ge=0, alias="newItemIndex")
    new_item_name: str | None = Field(default=None, alias="newItemName")
    reason: str | None = None
    estimated_savings_amount: Decimal | None = Field(
        default=None, ge=Decimal("0"), alias="estimatedSavingsAmount"
    )
    currency: CurrencyCode | None = None

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            return value.strip().upper() or None
        return value

    @field_serializer("estimated_savings_amount", when_used="json")
    def serialize_savings(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


class BudgetOptimizationPreservedItem(APIModel):
    item_index: int = Field(ge=0, alias="itemIndex")
    item_name: str = Field(alias="itemName")
    reason: str | None = None


class BudgetOptimizationProposalResponse(APIModel):
    summary: NonEmptyString
    scope: Literal["day"] = "day"
    day_number: int = Field(ge=1, alias="dayNumber")
    currency: CurrencyCode = "EUR"
    base_day_estimated_total: Decimal = Field(ge=Decimal("0"), alias="baseDayEstimatedTotal")
    proposed_day_estimated_total: Decimal = Field(
        ge=Decimal("0"), alias="proposedDayEstimatedTotal"
    )
    estimated_savings_amount: Decimal = Field(ge=Decimal("0"), alias="estimatedSavingsAmount")
    confidence: Literal["low", "medium", "high"] = "medium"
    changes: list[BudgetOptimizationChange] = Field(default_factory=list)
    preserved_items: list[BudgetOptimizationPreservedItem] = Field(
        default_factory=list, alias="preservedItems"
    )
    tradeoffs: list[str] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)
    proposed_day: ItineraryDay = Field(alias="proposedDay")

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper()
        return value

    @model_validator(mode="after")
    def proposal_day_must_match(self) -> "BudgetOptimizationProposalResponse":
        if self.proposed_day.day != self.day_number:
            raise ValueError("proposedDay.day must match dayNumber")
        if self.proposed_day_estimated_total > self.base_day_estimated_total:
            raise ValueError("proposedDayEstimatedTotal must not exceed baseDayEstimatedTotal")
        return self

    @field_serializer(
        "base_day_estimated_total",
        "proposed_day_estimated_total",
        "estimated_savings_amount",
        when_used="json",
    )
    def serialize_decimal(self, value: Decimal) -> int | float:
        serialized = _serialize_decimal(value)
        return 0 if serialized is None else serialized
