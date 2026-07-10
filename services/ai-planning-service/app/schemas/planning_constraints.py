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


class PreviousTripSignals(APIModel):
    visited_destinations: list[str] = Field(default_factory=list, alias="visitedDestinations")
    liked_styles: list[str] = Field(default_factory=list, alias="likedStyles")
    typical_duration_days: int | None = Field(default=None, alias="typicalDurationDays")
    typical_budget: PlanningConstraintBudget | None = Field(default=None, alias="typicalBudget")


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
    previous_trip_signals: PreviousTripSignals | None = Field(
        default=None,
        alias="previousTripSignals",
    )
    prompt: PlanningConstraintPrompt | None = None
    warnings: list[PlanningConstraintIssue] = Field(default_factory=list)
    blockers: list[PlanningConstraintIssue] = Field(default_factory=list)

