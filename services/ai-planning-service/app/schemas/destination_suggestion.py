from datetime import date
from enum import StrEnum
from typing import Literal

from pydantic import Field, field_validator, model_validator

from app.schemas.itinerary import APIModel, OutputLanguage, TripRoute, UserPreferences
from app.schemas.observability import AIResponseMetadata
from app.schemas.planning_constraints import PlanningConstraints


class DestinationSuggestionMode(StrEnum):
    PROMPT = "prompt"
    SURPRISE = "surprise"
    REFINE = "refine"


class DestinationUserContext(APIModel):
    home_city: str | None = Field(default=None, alias="homeCity", max_length=200)
    home_country: str | None = Field(default=None, alias="homeCountry", max_length=200)
    preferred_currency: str = Field(default="EUR", alias="preferredCurrency")
    preferred_language: OutputLanguage = Field(default="en", alias="preferredLanguage")
    preferences: UserPreferences | None = None

    @field_validator("preferred_currency")
    @classmethod
    def normalize_currency(cls, value: str) -> str:
        normalized = value.strip().upper()
        if len(normalized) != 3 or not normalized.isalpha():
            raise ValueError("preferredCurrency must be a 3-letter currency code")
        return normalized


class DestinationBudget(APIModel):
    amount: float = Field(ge=0)
    currency: str = "EUR"

    @field_validator("currency")
    @classmethod
    def normalize_currency(cls, value: str) -> str:
        normalized = value.strip().upper()
        if len(normalized) != 3 or not normalized.isalpha():
            raise ValueError("currency must be a 3-letter currency code")
        return normalized


class DestinationTripContext(APIModel):
    duration_days: int | None = Field(default=None, ge=1, le=30, alias="durationDays")
    start_date: date | None = Field(default=None, alias="startDate")
    date_flexibility: str | None = Field(default=None, alias="dateFlexibility", max_length=100)
    budget: DestinationBudget | None = None
    travelers: int = Field(default=1, ge=1, le=50)
    origin: str | None = Field(default=None, max_length=300)
    scope: Literal["personal", "workspace"] = "personal"


class PreviousTripSummary(APIModel):
    destination: str = Field(min_length=1, max_length=200)
    country: str | None = Field(default=None, max_length=200)
    duration_days: int = Field(ge=1, le=60, alias="durationDays")
    budget: DestinationBudget | None = None
    tags: list[str] = Field(default_factory=list, max_length=30)
    liked_signals: list[str] = Field(default_factory=list, alias="likedSignals", max_length=30)
    pace: str | None = Field(default=None, max_length=50)
    created_at: date | None = Field(default=None, alias="createdAt")


class DestinationRefinementContext(APIModel):
    previous_suggestions: list["DestinationSuggestion"] = Field(
        default_factory=list,
        alias="previousSuggestions",
        max_length=10,
    )
    selected_suggestion_id: str | None = Field(
        default=None,
        alias="selectedSuggestionId",
        max_length=200,
    )
    instruction: str = Field(default="", max_length=1000)


class DestinationSuggestionConstraints(APIModel):
    suggestion_count: int = Field(default=5, ge=3, le=5, alias="suggestionCount")
    avoid_previously_visited: bool = Field(default=True, alias="avoidPreviouslyVisited")
    prefer_novelty: bool = Field(default=True, alias="preferNovelty")
    include_reasoning: bool = Field(default=True, alias="includeReasoning")
    max_travel_complexity: Literal["low", "medium", "high"] = Field(
        default="medium",
        alias="maxTravelComplexity",
    )


class WorkspacePolicyConstraints(APIModel):
    summary: str = Field(default="", max_length=5000)
    rules: dict[str, object] = Field(default_factory=dict)


class DestinationSuggestionRequest(APIModel):
    prompt: str = Field(default="", max_length=1000)
    mode: DestinationSuggestionMode
    output_language: OutputLanguage = Field(default="en", alias="outputLanguage")
    user_context: DestinationUserContext | None = Field(default=None, alias="userContext")
    trip_context: DestinationTripContext | None = Field(default=None, alias="tripContext")
    previous_trips: list[PreviousTripSummary] = Field(
        default_factory=list,
        alias="previousTrips",
        max_length=20,
    )
    workspace_policy_constraints: WorkspacePolicyConstraints | None = Field(
        default=None,
        alias="workspacePolicyConstraints",
    )
    planning_constraints: PlanningConstraints | None = Field(
        default=None,
        alias="planningConstraints",
    )
    refinement: DestinationRefinementContext | None = None
    constraints: DestinationSuggestionConstraints = Field(
        default_factory=DestinationSuggestionConstraints
    )

    @field_validator("prompt")
    @classmethod
    def normalize_prompt(cls, value: str) -> str:
        return value.strip()

    @model_validator(mode="after")
    def validate_mode_context(self) -> "DestinationSuggestionRequest":
        if self.mode == DestinationSuggestionMode.PROMPT and not self.prompt:
            raise ValueError("prompt is required in prompt mode")
        if self.mode == DestinationSuggestionMode.REFINE and (
            self.refinement is None or not self.refinement.instruction.strip()
        ):
            raise ValueError("refinement.instruction is required in refine mode")
        return self


class DestinationBudgetEstimate(DestinationBudget):
    confidence: Literal["low", "medium", "high"] = "medium"


class DestinationTripPreview(APIModel):
    title: str = Field(min_length=1, max_length=300)
    summary: str = Field(min_length=1, max_length=1000)
    sample_day: list[str] = Field(
        default_factory=list,
        alias="sampleDay",
        min_length=1,
        max_length=8,
    )


class DestinationConcern(APIModel):
    type: str = Field(min_length=1, max_length=100)
    message: str = Field(min_length=1, max_length=500)


class DestinationSuggestion(APIModel):
    id: str = Field(min_length=1, max_length=200)
    suggestion_type: Literal["single_destination", "route"] = Field(
        default="single_destination", alias="suggestionType"
    )
    destination: str = Field(min_length=1, max_length=200)
    city: str = Field(min_length=1, max_length=200)
    country: str = Field(min_length=1, max_length=200)
    region: str | None = Field(default=None, max_length=200)
    match_score: int = Field(ge=0, le=100, alias="matchScore")
    recommended_duration_days: int = Field(ge=1, le=30, alias="recommendedDurationDays")
    best_for: list[str] = Field(default_factory=list, alias="bestFor", max_length=12)
    estimated_budget: DestinationBudgetEstimate = Field(alias="estimatedBudget")
    best_time_to_go: str = Field(alias="bestTimeToGo", min_length=1, max_length=300)
    why_it_fits: str = Field(alias="whyItFits", min_length=1, max_length=1000)
    why_this_fits_you: list[str] = Field(default_factory=list, alias="whyThisFitsYou", max_length=5)
    personalization_tags: list[str] = Field(
        default_factory=list, alias="personalizationTags", max_length=8
    )
    tradeoffs: list[str] = Field(default_factory=list, max_length=5)
    possible_downsides: list[str] = Field(
        default_factory=list,
        alias="possibleDownsides",
        max_length=8,
    )
    trip_preview: DestinationTripPreview = Field(alias="tripPreview")
    tags: list[str] = Field(default_factory=list, max_length=20)
    suggested_prompt_for_itinerary: str = Field(
        alias="suggestedPromptForItinerary",
        min_length=1,
        max_length=1000,
    )
    route: TripRoute | None = None
    concerns: list[DestinationConcern] = Field(default_factory=list, max_length=8)


class DestinationSuggestionResponse(APIModel):
    session_title: str = Field(alias="sessionTitle", min_length=1, max_length=300)
    suggestions: list[DestinationSuggestion] = Field(min_length=1, max_length=5)
    follow_up_questions: list[str] = Field(
        default_factory=list,
        alias="followUpQuestions",
        max_length=5,
    )
    warnings: list[str] = Field(default_factory=list, max_length=8)
    metadata: AIResponseMetadata | None = None
