from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class ObservabilityModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True)


class TokenEstimate(ObservabilityModel):
    prompt: int = Field(default=0, ge=0)
    completion: int = Field(default=0, ge=0)
    total: int = Field(default=0, ge=0)


class AIResponseMetadata(ObservabilityModel):
    prompt_version: str = Field(alias="promptVersion")
    provider: str
    model: str | None = None
    mode: str
    duration_ms: int = Field(alias="durationMs", ge=0)
    token_estimate: TokenEstimate = Field(alias="tokenEstimate")


class PromptBuildMetadata(ObservabilityModel):
    prompt_version: str = Field(alias="promptVersion")
    builder: str
    sections: list[str] = Field(default_factory=list)
    char_count: int = Field(alias="charCount", ge=0)
    token_estimate: int = Field(alias="tokenEstimate", ge=0)
    rag_chunk_count: int = Field(default=0, alias="ragChunkCount", ge=0)
    redaction_applied: bool = Field(default=True, alias="redactionApplied")


class PromptBuildResult(ObservabilityModel):
    prompt: str
    metadata: PromptBuildMetadata


class RAGRetrievalMetadata(ObservabilityModel):
    enabled: bool
    collection_name: str | None = Field(default=None, alias="collectionName")
    retrieved_chunk_count: int = Field(default=0, alias="retrievedChunkCount", ge=0)
    suspicious_prompt_injection_warning_count: int = Field(
        default=0, alias="suspiciousPromptInjectionWarningCount", ge=0
    )
