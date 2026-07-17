from __future__ import annotations

from typing import Any

from pydantic import Field

from app.schemas.itinerary import APIModel
from app.schemas.observability import AIResponseMetadata


class GenerationValidationIssue(APIModel):
    id: str
    category: str
    severity: str
    title: str
    description: str | None = None
    fixability: str | None = None
    day_number: int | None = Field(default=None, alias="dayNumber")
    item_index: int | None = Field(default=None, alias="itemIndex")
    route_leg_id: str | None = Field(default=None, alias="routeLegId")
    rule_key: str | None = Field(default=None, alias="ruleKey")


class RepairScope(APIModel):
    type: str
    day_number: int | None = Field(default=None, alias="dayNumber")
    item_index: int | None = Field(default=None, alias="itemIndex")
    route_leg_id: str | None = Field(default=None, alias="routeLegId")


class RepairConstraints(APIModel):
    preserve_unaffected_days: bool = Field(default=True, alias="preserveUnaffectedDays")
    preserve_user_edited_items: bool = Field(default=True, alias="preserveUserEditedItems")
    output_language: str = Field(default="en", alias="outputLanguage")


class RepairPlanningContext(APIModel):
    trip: dict[str, Any] = Field(default_factory=dict)
    route: dict[str, Any] | None = None
    accommodation: dict[str, Any] | None = None
    planning_constraints: dict[str, Any] | None = Field(
        default=None,
        alias="planningConstraints",
    )
    weather_forecast: dict[str, Any] | None = Field(default=None, alias="weatherForecast")
    budget_summary: dict[str, Any] | None = Field(default=None, alias="budgetSummary")
    workspace_policy: dict[str, Any] | None = Field(default=None, alias="workspacePolicy")


class RepairGenerationOutputRequest(APIModel):
    generation_type: str = Field(alias="generationType")
    current_output: dict[str, Any] = Field(alias="currentOutput")
    validation_issues: list[GenerationValidationIssue] = Field(
        default_factory=list,
        alias="validationIssues",
    )
    planning_context: RepairPlanningContext = Field(alias="planningContext")
    repair_scope: RepairScope = Field(alias="repairScope")
    constraints: RepairConstraints = Field(default_factory=RepairConstraints)


class GenerationRepairChange(APIModel):
    type: str
    description: str | None = None
    day_number: int | None = Field(default=None, alias="dayNumber")
    item_index: int | None = Field(default=None, alias="itemIndex")
    metadata: dict[str, Any] | None = None


class RepairGenerationOutputResponse(APIModel):
    repaired_output: dict[str, Any] = Field(alias="repairedOutput")
    changes_made: list[GenerationRepairChange] = Field(
        default_factory=list,
        alias="changesMade",
    )
    warnings: list[str] = Field(default_factory=list)
    metadata: AIResponseMetadata | None = None
