from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator


class APIModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True)


ConstraintSeverity = Literal["info", "warning", "blocking"]
PlanningSource = Literal[
    "trip_discovery",
    "trip_generation",
    "day_regeneration",
    "item_regeneration",
    "template_adaptation",
    "policy_repair",
    "budget_optimization",
    "route_generation",
    "route_alternatives",
    "route_alternative_refinement",
    "route_alternative_apply_preview",
    "route_update_preview",
]


class ConstraintSuggestedAction(APIModel):
    type: str
    label: str


class PlanningConstraintIssue(APIModel):
    type: str
    severity: ConstraintSeverity
    message: str
    source: str = ""
    affected: dict[str, Any] | None = None
    suggested_actions: list[ConstraintSuggestedAction] = Field(
        default_factory=list,
        alias="suggestedActions",
    )


class PlanningConstraintBudget(APIModel):
    amount: float | None = None
    currency: str = "EUR"
    strictness: Literal["loose", "target", "strict"] = "target"

    @field_validator("currency", mode="before")
    @classmethod
    def normalize_currency(cls, value: object) -> object:
        if value is None:
            return "EUR"
        if isinstance(value, str):
            return value.strip().upper() or "EUR"
        return value


class PlanningConstraintDates(APIModel):
    start_date: str | None = Field(default=None, alias="startDate")
    end_date: str | None = Field(default=None, alias="endDate")
    duration_days: int | None = Field(default=None, alias="durationDays")
    flexibility: Literal["fixed", "flexible", "weekend", "month", "unknown"] = "fixed"


class PlanningConstraintTravelers(APIModel):
    count: int = 1
    type: str | None = None


class PlanningConstraintWalking(APIModel):
    max_km_per_day: float | None = Field(default=None, alias="maxKmPerDay")
    allow_long_hikes: bool = Field(default=True, alias="allowLongHikes")


class PlanningConstraintTransport(APIModel):
    preferred_modes: list[str] = Field(default_factory=list, alias="preferredModes")
    allowed_modes: list[str] = Field(default_factory=list, alias="allowedModes")
    avoid_modes: list[str] = Field(default_factory=list, alias="avoidModes")
    disallowed_modes: list[str] = Field(default_factory=list, alias="disallowedModes")
    car_available: bool = Field(default=False, alias="carAvailable")
    max_transfer_hours_per_day: int | None = Field(
        default=None,
        alias="maxTransferHoursPerDay",
    )


class PlanningConstraintAccommodation(APIModel):
    preferred_types: list[str] = Field(default_factory=list, alias="preferredTypes")
    avoid_types: list[str] = Field(default_factory=list, alias="avoidTypes")
    camping_allowed: bool = Field(default=False, alias="campingAllowed")


class PlanningConstraintAccessibility(APIModel):
    low_walking_required: bool = Field(default=False, alias="lowWalkingRequired")
    step_free_preferred: bool = Field(default=False, alias="stepFreePreferred")
    notes: str | None = None


class PlanningConstraintFood(APIModel):
    preferences: list[str] = Field(default_factory=list)
    dietary_restrictions: list[str] = Field(default_factory=list, alias="dietaryRestrictions")


class PlanningConstraintWorkspacePolicy(APIModel):
    policy_id: str | None = Field(default=None, alias="policyId")
    summary: str | None = None
    blocking_rules: list[str] = Field(default_factory=list, alias="blockingRules")
    warning_rules: list[str] = Field(default_factory=list, alias="warningRules")
    rules: dict[str, Any] | None = None


class GroupPreferenceItem(APIModel):
    day_number: int = Field(alias="dayNumber")
    item_index: int = Field(alias="itemIndex")
    item_id: str | None = Field(default=None, alias="itemId")
    name: str = ""
    count: int = 0
    score: int = 0


class GroupRouteAlternativeVote(APIModel):
    session_id: str = Field(default="", alias="sessionId")
    alternative_id: str = Field(default="", alias="alternativeId")
    label: str = ""
    score: int = 0
    votes: int = 0


class PlanningConstraintGroupPreferences(APIModel):
    summary: str = ""
    must_have_items: list[GroupPreferenceItem] = Field(
        default_factory=list,
        alias="mustHaveItems",
    )
    skip_candidates: list[GroupPreferenceItem] = Field(
        default_factory=list,
        alias="skipCandidates",
    )
    preferred_destinations: list[str] = Field(
        default_factory=list,
        alias="preferredDestinations",
    )
    preferred_transport_modes: list[str] = Field(
        default_factory=list,
        alias="preferredTransportModes",
    )
    preferred_dates: list[str] = Field(default_factory=list, alias="preferredDates")
    preferred_route_alternative_id: str | None = Field(
        default=None,
        alias="preferredRouteAlternativeId",
    )
    preferred_route_session_id: str | None = Field(
        default=None,
        alias="preferredRouteSessionId",
    )
    route_alternative_votes: list[GroupRouteAlternativeVote] = Field(
        default_factory=list,
        alias="routeAlternativeVotes",
    )
    open_decision_count: int = Field(default=0, alias="openDecisionCount")


class SelectedDateOption(APIModel):
    start_date: str = Field(alias="startDate")
    end_date: str = Field(alias="endDate")
    duration_days: int = Field(alias="durationDays")
    score: int = 0
    conflict_user_count: int = Field(default=0, alias="conflictUserCount")


class PlanningConstraintGroupAvailability(APIModel):
    submitted_count: int = Field(default=0, alias="submittedCount")
    total_collaborators: int = Field(default=0, alias="totalCollaborators")
    selected_date_option: SelectedDateOption | None = Field(
        default=None,
        alias="selectedDateOption",
    )
    missing_response_count: int = Field(default=0, alias="missingResponseCount")
    notes: str | None = None


class PreviousTripSignals(APIModel):
    visited_destinations: list[str] = Field(default_factory=list, alias="visitedDestinations")
    liked_styles: list[str] = Field(default_factory=list, alias="likedStyles")
    typical_duration_days: int | None = Field(default=None, alias="typicalDurationDays")
    typical_budget: PlanningConstraintBudget | None = Field(default=None, alias="typicalBudget")


class PersonalizationMoney(APIModel):
    amount: float
    currency: str = "EUR"


class PersonalizationPastTripSignals(APIModel):
    past_destination_count: int = Field(default=0, alias="pastDestinationCount")
    recent_destinations: list[str] = Field(default_factory=list, alias="recentDestinations")
    repeated_styles: list[str] = Field(default_factory=list, alias="repeatedStyles")
    average_trip_duration_days: int | None = Field(default=None, alias="averageTripDurationDays")
    average_budget_per_day: PersonalizationMoney | None = Field(
        default=None, alias="averageBudgetPerDay"
    )
    preferred_transport_from_history: list[str] = Field(
        default_factory=list, alias="preferredTransportFromHistory"
    )
    over_budget_pattern: bool = Field(default=False, alias="overBudgetPattern")


class PersonalizationFeedbackSignals(APIModel):
    liked_destinations: list[str] = Field(default_factory=list, alias="likedDestinations")
    disliked_destinations: list[str] = Field(default_factory=list, alias="dislikedDestinations")
    liked_styles: list[str] = Field(default_factory=list, alias="likedStyles")
    disliked_styles: list[str] = Field(default_factory=list, alias="dislikedStyles")
    too_expensive_count: int = Field(default=0, alias="tooExpensiveCount")
    too_much_walking_count: int = Field(default=0, alias="tooMuchWalkingCount")
    prefer_train_count: int = Field(default=0, alias="preferTrainCount")
    budget_sensitivity: str = Field(default="medium", alias="budgetSensitivity")
    walking_sensitivity: str = Field(default="moderate", alias="walkingSensitivity")
    recent_feedback_count: int = Field(default=0, alias="recentFeedbackCount")


class PersonalizationSummary(APIModel):
    schema_version: str = Field(default="personalization_v2", alias="schemaVersion")
    completeness_score: int = Field(default=0, ge=0, le=100, alias="completenessScore")
    travel_styles: list[str] = Field(default_factory=list, alias="travelStyles")
    transport_bias: list[str] = Field(default_factory=list, alias="transportBias")
    activity_bias: list[str] = Field(default_factory=list, alias="activityBias")
    avoid_bias: list[str] = Field(default_factory=list, alias="avoidBias")
    budget_comfort: str = Field(default="medium", alias="budgetComfort")
    walking_tolerance: str = Field(default="moderate", alias="walkingTolerance")
    past_trip_signals: PersonalizationPastTripSignals = Field(
        default_factory=PersonalizationPastTripSignals, alias="pastTripSignals"
    )
    feedback_signals: PersonalizationFeedbackSignals = Field(
        default_factory=PersonalizationFeedbackSignals, alias="feedbackSignals"
    )
    explanation_inputs: list[str] = Field(default_factory=list, alias="explanationInputs")


class PlanningConstraintPrompt(APIModel):
    user_prompt: str | None = Field(default=None, alias="userPrompt")
    quick_chips: list[str] = Field(default_factory=list, alias="quickChips")
    refinement_instruction: str | None = Field(default=None, alias="refinementInstruction")


class PlanningConstraints(APIModel):
    schema_version: int = Field(default=1, alias="schemaVersion")
    language: str = "en"
    scope: Literal["personal", "workspace"] = "personal"
    workspace_id: str | None = Field(default=None, alias="workspaceId")
    source: PlanningSource
    profile: dict[str, Any] = Field(default_factory=dict)
    budget: PlanningConstraintBudget | None = None
    dates: PlanningConstraintDates = Field(default_factory=PlanningConstraintDates)
    travelers: PlanningConstraintTravelers = Field(default_factory=PlanningConstraintTravelers)
    pace: Literal["relaxed", "balanced", "packed"] = "balanced"
    walking: PlanningConstraintWalking = Field(default_factory=PlanningConstraintWalking)
    transport: PlanningConstraintTransport = Field(default_factory=PlanningConstraintTransport)
    trip_styles: list[str] = Field(default_factory=list, alias="tripStyles")
    accommodation: PlanningConstraintAccommodation = Field(
        default_factory=PlanningConstraintAccommodation
    )
    interests: list[str] = Field(default_factory=list)
    avoid: list[str] = Field(default_factory=list)
    must_have: list[str] = Field(default_factory=list, alias="mustHave")
    accessibility: PlanningConstraintAccessibility = Field(
        default_factory=PlanningConstraintAccessibility
    )
    food: PlanningConstraintFood = Field(default_factory=PlanningConstraintFood)
    route: dict[str, Any] | None = None
    workspace_policy: PlanningConstraintWorkspacePolicy | None = Field(
        default=None,
        alias="workspacePolicy",
    )
    group_preferences: PlanningConstraintGroupPreferences | None = Field(
        default=None,
        alias="groupPreferences",
    )
    group_availability: PlanningConstraintGroupAvailability | None = Field(
        default=None,
        alias="groupAvailability",
    )
    previous_trip_signals: PreviousTripSignals | None = Field(
        default=None,
        alias="previousTripSignals",
    )
    personalization: PersonalizationSummary | None = None
    prompt: PlanningConstraintPrompt | None = None
    warnings: list[PlanningConstraintIssue] = Field(default_factory=list)
    blockers: list[PlanningConstraintIssue] = Field(default_factory=list)
