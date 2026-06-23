from datetime import date
from decimal import Decimal
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


def _serialize_decimal(value: Decimal | None) -> int | float | None:
    if value is None:
        return None
    if value == value.to_integral_value():
        return int(value)
    return float(value)


class APIModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True)


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
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    weather_forecast: WeatherForecast | None = Field(default=None, alias="weatherForecast")

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


class ItineraryItem(APIModel):
    time: str
    type: str
    name: str
    note: str | None = None
    estimated_cost: Decimal | None = Field(default=None, alias="estimatedCost")

    @field_serializer("estimated_cost", when_used="json")
    def serialize_estimated_cost(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


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
    user_profile: UserProfile | None = Field(default=None, alias="userProfile")
    user_preferences: UserPreferences | None = Field(default=None, alias="userPreferences")
    weather_forecast: WeatherForecast | None = Field(default=None, alias="weatherForecast")

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
