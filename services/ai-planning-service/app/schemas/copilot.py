# ruff: noqa: E501
from __future__ import annotations

import json
from typing import Any, Literal

from pydantic import Field, field_validator, model_validator

from app.schemas.itinerary import APIModel, OutputLanguage

CopilotIntent = Literal[
    "next_action",
    "explain_health",
    "explain_budget",
    "explain_route",
    "explain_group_readiness",
    "explain_checklist",
    "explain_expenses",
    "explain_approval",
    "explain_feature",
    "how_to",
    "find_section",
    "unsafe_mutation_request",
    "out_of_scope",
    "general_trip_question",
]
CopilotActionType = Literal[
    "open_command_center",
    "open_trip_health",
    "open_route",
    "open_route_leg",
    "find_transport",
    "open_budget",
    "open_budget_confidence",
    "open_expenses",
    "upload_receipt",
    "add_expense",
    "open_checklist",
    "generate_checklist_screen",
    "open_reminders",
    "open_group_readiness",
    "request_availability_screen",
    "open_polls",
    "open_approval",
    "open_policy",
    "open_itinerary",
    "open_itinerary_day",
    "open_generation_quality",
    "open_version_history",
    "open_share_settings",
    "open_offline_settings",
    "open_notification_settings",
    "open_settings",
    "open_search",
    "no_action",
]
CopilotSourceType = Literal[
    "command_center",
    "trip_health",
    "budget_confidence",
    "group_readiness",
    "route_summary",
    "itinerary_summary",
    "checklist_summary",
    "reminders_summary",
    "expenses_summary",
    "approval_status",
    "policy_evaluation",
    "generation_quality",
    "personalization",
    "notification_summary",
    "app_help",
    "unknown",
]


class CopilotAvailableAction(APIModel):
    type: CopilotActionType
    label: str = Field(min_length=1, max_length=120)
    href: str = Field(min_length=1, max_length=512)
    style: Literal["primary", "secondary"] = "secondary"

    @field_validator("href")
    @classmethod
    def require_trip_relative_href(cls, value: str) -> str:
        value = value.strip()
        if not value.startswith("/trips/") or "://" in value or "\\" in value:
            raise ValueError("action href must be a relative trip URL")
        return value


class CopilotPermissionSummary(APIModel):
    role: Literal["owner", "editor", "viewer", "none"]
    can_edit_itinerary: bool = Field(alias="canEditItinerary")
    can_edit_route: bool = Field(alias="canEditRoute")
    can_manage_share: bool = Field(alias="canManageShare")
    can_upload_receipt: bool = Field(alias="canUploadReceipt")
    can_comment: bool = Field(alias="canComment")
    can_vote: bool = Field(alias="canVote")


class CopilotSafeContext(APIModel):
    data: dict[str, Any] = Field(alias="safeContext")

    @field_validator("data")
    @classmethod
    def enforce_safe_context_size(cls, value: dict[str, Any]) -> dict[str, Any]:
        if len(json.dumps(value, ensure_ascii=False, separators=(",", ":"))) > 50_000:
            raise ValueError("safeContext is too large")
        return value


class CopilotSuggestedAction(APIModel):
    type: CopilotActionType
    label: str = Field(min_length=1, max_length=120)
    href: str = Field(min_length=1, max_length=512)


class CopilotRespondRequest(APIModel):
    message: str = Field(min_length=1, max_length=2000)
    language: OutputLanguage = "en"
    intent: CopilotIntent
    safe_context: dict[str, Any] = Field(alias="safeContext")
    available_actions: list[CopilotAvailableAction] = Field(
        default_factory=list, max_length=40, alias="availableActions"
    )
    permission_summary: CopilotPermissionSummary = Field(alias="permissionSummary")
    conversation_summary: str | None = Field(default=None, max_length=500, alias="conversationSummary")

    @field_validator("message")
    @classmethod
    def normalize_message(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("message cannot be blank")
        return value

    @model_validator(mode="after")
    def enforce_safe_context_size(self) -> CopilotRespondRequest:
        if len(json.dumps(self.safe_context, ensure_ascii=False, separators=(",", ":"))) > 50_000:
            raise ValueError("safeContext is too large")
        return self


class CopilotRespondResponse(APIModel):
    answer: str = Field(min_length=1, max_length=2400)
    actions: list[CopilotSuggestedAction] = Field(default_factory=list, max_length=2)
    source_types: list[CopilotSourceType] = Field(default_factory=list, max_length=4, alias="sourceTypes")
    warnings: list[str] = Field(default_factory=list, max_length=3)

    @field_validator("warnings", mode="before")
    @classmethod
    def clean_warnings(cls, value: object) -> object:
        if not isinstance(value, list):
            return []
        return [item.strip()[:240] for item in value if isinstance(item, str) and item.strip()][:3]
