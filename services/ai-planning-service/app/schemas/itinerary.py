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
    items: list[ItineraryItem] = Field(min_length=3)


class ItineraryResponse(APIModel):
    days: list[ItineraryDay]
