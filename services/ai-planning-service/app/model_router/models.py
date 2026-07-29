from __future__ import annotations

from enum import StrEnum
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, field_validator


class RouterModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True, protected_namespaces=())


class DeploymentStatus(StrEnum):
    REGISTERED = "registered"
    CANDIDATE = "candidate"
    SHADOW = "shadow"
    INTERNAL = "internal"
    ALLOWLIST = "allowlist"
    STAGED_ROLLOUT = "staged_rollout"
    ACTIVE = "active"
    PAUSED = "paused"
    REJECTED = "rejected"
    RETIRED = "retired"


class ModelVariant(StrEnum):
    GROUNDED_BASELINE = "grounded_baseline"
    FINE_TUNED_CANDIDATE = "fine_tuned_candidate"


class TrafficMode(StrEnum):
    DISABLED = "disabled"
    SHADOW = "shadow"
    INTERNAL = "internal"
    ALLOWLIST = "allowlist"
    PERCENTAGE = "percentage"
    ACTIVE = "active"


class InferenceMode(StrEnum):
    PRIMARY = "primary"
    SHADOW = "shadow"
    EVALUATION = "evaluation"


class ModelRoutingRequestMetadata(RouterModel):
    deployment_key: str | None = Field(default=None, alias="deploymentKey", max_length=120)
    request_assignment_id: UUID | None = Field(default=None, alias="requestAssignmentId")
    inference_mode: InferenceMode | None = Field(default=None, alias="inferenceMode")

    @field_validator("deployment_key")
    @classmethod
    def normalize_deployment_key(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip()
        if not normalized:
            return None
        if "/" in normalized or "\\" in normalized or ".." in normalized:
            raise ValueError("deploymentKey must be a registered deployment key, not a path")
        for char in normalized:
            if not (char.isalnum() or char in {"-", "_", ".", ":"}):
                raise ValueError("deploymentKey contains unsupported characters")
        return normalized


class RuntimeDeploymentMetadata(RouterModel):
    deployment_key: str | None = Field(default=None, alias="deploymentKey")
    request_assignment_id: UUID | None = Field(default=None, alias="requestAssignmentId")
    inference_mode: InferenceMode = Field(default=InferenceMode.PRIMARY, alias="inferenceMode")
    model_variant: ModelVariant = Field(default=ModelVariant.GROUNDED_BASELINE, alias="modelVariant")
