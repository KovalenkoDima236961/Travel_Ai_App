from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field


class ObservabilityModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True, protected_namespaces=())


class TokenEstimate(ObservabilityModel):
    prompt: int = Field(default=0, ge=0)
    completion: int = Field(default=0, ge=0)
    total: int = Field(default=0, ge=0)


class AIResponseMetadata(ObservabilityModel):
    prompt_version: str = Field(alias="promptVersion")
    provider: str
    model: str | None = None
    mode: str
    provider_request_id: str | None = Field(default=None, alias="providerRequestId")
    deployment_key: str | None = Field(default=None, alias="deploymentKey")
    request_assignment_id: UUID | None = Field(default=None, alias="requestAssignmentId")
    inference_mode: str | None = Field(default=None, alias="inferenceMode")
    model_variant: str | None = Field(default=None, alias="modelVariant")
    adapter_enabled: bool | None = Field(default=None, alias="adapterEnabled")
    adapter_loaded: bool | None = Field(default=None, alias="adapterLoaded")
    adapter_key: str | None = Field(default=None, alias="adapterKey")
    adapter_checksum: str | None = Field(default=None, alias="adapterChecksum")
    adapter_checksum_verified: bool | None = Field(
        default=None,
        alias="adapterChecksumVerified",
    )
    experiment_key: str | None = Field(default=None, alias="experimentKey")
    dataset_version: str | None = Field(default=None, alias="datasetVersion")
    fallback_to_base: bool | None = Field(default=None, alias="fallbackToBase")
    fallback_used: bool | None = Field(default=None, alias="fallbackUsed")
    fallback_provider: str | None = Field(default=None, alias="fallbackProvider")
    quality_status: str | None = Field(default=None, alias="qualityStatus")
    needs_review: bool | None = Field(default=None, alias="needsReview")
    duration_ms: int = Field(alias="durationMs", ge=0)
    token_estimate: TokenEstimate = Field(alias="tokenEstimate")
    input_tokens: int | None = Field(default=None, alias="inputTokens", ge=0)
    output_tokens: int | None = Field(default=None, alias="outputTokens", ge=0)
    total_tokens: int | None = Field(default=None, alias="totalTokens", ge=0)
    cached_input_tokens: int | None = Field(default=None, alias="cachedInputTokens", ge=0)
    reasoning_tokens: int | None = Field(default=None, alias="reasoningTokens", ge=0)
    retry_count: int | None = Field(default=None, alias="retryCount", ge=0)


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
