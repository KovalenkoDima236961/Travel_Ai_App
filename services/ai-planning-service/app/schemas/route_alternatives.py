from __future__ import annotations

from datetime import date
from decimal import Decimal
from typing import Literal

from pydantic import Field, field_serializer, field_validator, model_validator

from app.schemas.itinerary import (
    APIModel,
    CurrencyCode,
    OutputLanguage,
    RoutePlace,
    TripRoute,
    _serialize_decimal,
)
from app.schemas.observability import AIResponseMetadata
from app.schemas.planning_constraints import PlanningConstraints

RouteAlternativeDifficulty = Literal["relaxed", "balanced", "intense", "rushed"]
BudgetConfidence = Literal["low", "medium", "high"]


class RouteAlternativeBudgetEstimate(APIModel):
    amount: Decimal | None = Field(default=None, ge=Decimal("0"))
    currency: CurrencyCode = "EUR"
    confidence: BudgetConfidence = "medium"

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper() or "EUR"
        return value

    @field_serializer("amount", when_used="json")
    def serialize_amount(self, value: Decimal | None) -> int | float | None:
        return _serialize_decimal(value)


class RouteAlternativeBudget(RouteAlternativeBudgetEstimate):
    pass


class RouteAlternativeScores(APIModel):
    overall_fit: int = Field(default=70, alias="overallFit")
    budget_fit: int = Field(default=70, alias="budgetFit")
    time_efficiency: int = Field(default=70, alias="timeEfficiency")
    relaxation: int = 70
    nature: int = 70
    culture: int = 70
    transport_simplicity: int = Field(default=70, alias="transportSimplicity")
    policy_compliance: int = Field(default=100, alias="policyCompliance")

    @field_validator(
        "overall_fit",
        "budget_fit",
        "time_efficiency",
        "relaxation",
        "nature",
        "culture",
        "transport_simplicity",
        "policy_compliance",
        mode="before",
    )
    @classmethod
    def clamp_score(cls, value: object) -> int:
        try:
            score = int(value)
        except (TypeError, ValueError):
            score = 70
        return min(100, max(0, score))


class RouteAlternativePersonalizationFit(APIModel):
    score: int = Field(default=70, ge=0, le=100)
    reasons: list[str] = Field(default_factory=list, max_length=5)
    concerns: list[str] = Field(default_factory=list, max_length=5)


class RouteAlternative(APIModel):
    id: str = Field(min_length=1, max_length=100)
    title: str = Field(min_length=1, max_length=160)
    summary: str = Field(default="", max_length=800)
    route: TripRoute
    scores: RouteAlternativeScores = Field(default_factory=RouteAlternativeScores)
    estimated_budget: RouteAlternativeBudgetEstimate | None = Field(
        default=None,
        alias="estimatedBudget",
    )
    estimated_transfer_minutes: int | None = Field(
        default=None,
        ge=0,
        alias="estimatedTransferMinutes",
    )
    estimated_transfer_cost: RouteAlternativeBudgetEstimate | None = Field(
        default=None,
        alias="estimatedTransferCost",
    )
    difficulty: RouteAlternativeDifficulty = "balanced"
    best_for: list[str] = Field(default_factory=list, max_length=8, alias="bestFor")
    pros: list[str] = Field(default_factory=list, max_length=8)
    cons: list[str] = Field(default_factory=list, max_length=8)
    warnings: list[str] = Field(default_factory=list, max_length=8)
    suggested_itinerary_prompt: str | None = Field(
        default=None,
        max_length=1000,
        alias="suggestedItineraryPrompt",
    )
    personalization_fit: RouteAlternativePersonalizationFit | None = Field(
        default=None, alias="personalizationFit"
    )

    @field_validator("id", mode="before")
    @classmethod
    def normalize_id(cls, value: object) -> object:
        if isinstance(value, str):
            return value.strip()
        return value

    @field_validator("best_for", "pros", "cons", "warnings", mode="before")
    @classmethod
    def clean_string_lists(cls, value: object) -> object:
        if value is None:
            return []
        if not isinstance(value, list):
            return value
        cleaned: list[str] = []
        for item in value:
            if not isinstance(item, str):
                continue
            text = item.strip()
            if text:
                cleaned.append(text[:300])
        return cleaned[:8]

    @model_validator(mode="after")
    def validate_route_shape(self) -> RouteAlternative:
        if len(self.route.stops) < 1:
            raise ValueError("route.stops must contain at least one stop")
        valid_ids = {"origin", *(stop.id for stop in self.route.stops)}
        stop_ids = {stop.id for stop in self.route.stops}
        for leg in self.route.legs:
            if leg.from_stop_id not in valid_ids:
                raise ValueError("route leg fromStopId must reference origin or a stop")
            if leg.to_stop_id not in stop_ids:
                raise ValueError("route leg toStopId must reference a stop")
        return self


class RouteAlternativeRefinement(APIModel):
    previous_alternatives: list[RouteAlternative] = Field(
        default_factory=list,
        max_length=5,
        alias="previousAlternatives",
    )
    instruction: str | None = Field(default=None, max_length=1000)
    selected_alternative_id: str | None = Field(default=None, alias="selectedAlternativeId")

    @field_validator("instruction", "selected_alternative_id", mode="before")
    @classmethod
    def normalize_optional_string(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class RouteAlternativeRequest(APIModel):
    origin: RoutePlace | None = None
    prompt: str | None = Field(default=None, max_length=2000)
    duration_days: int | None = Field(default=None, ge=1, le=30, alias="durationDays")
    start_date: date | None = Field(default=None, alias="startDate")
    budget: RouteAlternativeBudget | None = None
    travelers: int = Field(default=1, ge=1, le=50)
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    planning_constraints: PlanningConstraints | None = Field(
        default=None,
        alias="planningConstraints",
    )
    current_route: TripRoute | None = Field(default=None, alias="currentRoute")
    refinement: RouteAlternativeRefinement = Field(default_factory=RouteAlternativeRefinement)
    suggestion_count: int = Field(default=3, ge=1, le=5, alias="suggestionCount")

    @field_validator("prompt", mode="before")
    @classmethod
    def normalize_prompt(cls, value: object) -> object:
        if value is None:
            return None
        if isinstance(value, str):
            trimmed = value.strip()
            return trimmed or None
        return value


class RouteAlternativeComparisonSummary(APIModel):
    cheapest_alternative_id: str | None = Field(default=None, alias="cheapestAlternativeId")
    most_relaxed_alternative_id: str | None = Field(
        default=None,
        alias="mostRelaxedAlternativeId",
    )
    best_nature_alternative_id: str | None = Field(default=None, alias="bestNatureAlternativeId")
    best_overall_alternative_id: str | None = Field(
        default=None,
        alias="bestOverallAlternativeId",
    )


class RouteAlternativeWarning(APIModel):
    type: str = "estimate_uncertainty"
    message: str


class RouteAlternativeResponse(APIModel):
    session_title: str = Field(alias="sessionTitle", min_length=1, max_length=160)
    alternatives: list[RouteAlternative] = Field(min_length=1, max_length=5)
    comparison_summary: RouteAlternativeComparisonSummary = Field(
        default_factory=RouteAlternativeComparisonSummary,
        alias="comparisonSummary",
    )
    follow_up_questions: list[str] = Field(
        default_factory=list,
        max_length=5,
        alias="followUpQuestions",
    )
    warnings: list[str] = Field(default_factory=list, max_length=8)
    metadata: AIResponseMetadata | None = None

    @field_validator("follow_up_questions", "warnings", mode="before")
    @classmethod
    def clean_lists(cls, value: object) -> object:
        if value is None:
            return []
        if not isinstance(value, list):
            return value
        return [item.strip()[:300] for item in value if isinstance(item, str) and item.strip()][:8]

    @model_validator(mode="after")
    def validate_unique_alternatives(self) -> RouteAlternativeResponse:
        ids = [alternative.id for alternative in self.alternatives]
        if len(ids) != len(set(ids)):
            raise ValueError("alternative ids must be unique")
        return self
