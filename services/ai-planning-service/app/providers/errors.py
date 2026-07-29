from __future__ import annotations

from enum import StrEnum
from typing import Any


class AIProviderErrorCode(StrEnum):
    DISABLED = "ai_provider_disabled"
    CONFIGURATION_INVALID = "ai_provider_configuration_invalid"
    AUTHENTICATION_FAILED = "ai_provider_authentication_failed"
    PERMISSION_DENIED = "ai_provider_permission_denied"
    RATE_LIMITED = "ai_provider_rate_limited"
    QUOTA_EXCEEDED = "ai_provider_quota_exceeded"
    TIMEOUT = "ai_provider_timeout"
    CONNECTION_FAILED = "ai_provider_connection_failed"
    UNAVAILABLE = "ai_provider_unavailable"
    INVALID_RESPONSE = "ai_provider_invalid_response"
    SCHEMA_VALIDATION_FAILED = "ai_provider_schema_validation_failed"
    CONTENT_REFUSED = "ai_provider_content_refused"
    CONTEXT_TOO_LARGE = "ai_provider_context_too_large"
    OUTPUT_LIMIT_EXCEEDED = "ai_provider_output_limit_exceeded"
    REPAIR_FAILED = "ai_provider_repair_failed"
    BUDGET_EXCEEDED = "ai_provider_budget_exceeded"
    CONCURRENCY_LIMIT = "ai_provider_concurrency_limit"
    UNKNOWN = "ai_provider_unknown_error"


class AIProviderError(RuntimeError):
    def __init__(
        self,
        code: AIProviderErrorCode,
        message: str,
        *,
        provider: str | None = None,
        model: str | None = None,
        request_id: str | None = None,
        retryable: bool = False,
        retry_after_seconds: int | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.provider = provider
        self.model = model
        self.request_id = request_id
        self.retryable = retryable
        self.retry_after_seconds = retry_after_seconds
        self.metadata = metadata or {}

    def safe_metadata(self) -> dict[str, Any]:
        return {
            key: value
            for key, value in {
                "provider": self.provider,
                "model": self.model,
                "requestId": self.request_id,
                "retryable": self.retryable,
                "retryAfterSeconds": self.retry_after_seconds,
                "errorCode": self.code.value,
                **self.metadata,
            }.items()
            if value is not None
        }


def normalize_openai_error(exc: BaseException, *, model: str | None = None) -> AIProviderError:
    request_id = getattr(exc, "request_id", None)
    status_code = getattr(exc, "status_code", None)
    name = type(exc).__name__
    message = "OpenAI provider request failed"
    retryable = False
    retry_after = _retry_after_seconds(exc)

    quota_marker = _error_text(exc).lower()

    if name == "AuthenticationError" or status_code == 401:
        code = AIProviderErrorCode.AUTHENTICATION_FAILED
    elif name == "PermissionDeniedError" or status_code == 403:
        code = AIProviderErrorCode.PERMISSION_DENIED
    elif name == "RateLimitError" or status_code == 429:
        code = (
            AIProviderErrorCode.QUOTA_EXCEEDED
            if "quota" in quota_marker or "insufficient_quota" in quota_marker
            else AIProviderErrorCode.RATE_LIMITED
        )
        retryable = True
    elif name == "APITimeoutError" or "timeout" in name.lower():
        code = AIProviderErrorCode.TIMEOUT
        retryable = True
    elif name == "APIConnectionError":
        code = AIProviderErrorCode.CONNECTION_FAILED
        retryable = True
    elif isinstance(status_code, int) and status_code >= 500:
        code = AIProviderErrorCode.UNAVAILABLE
        retryable = True
    elif status_code == 400:
        code = AIProviderErrorCode.CONFIGURATION_INVALID
    elif status_code == 422:
        code = AIProviderErrorCode.SCHEMA_VALIDATION_FAILED
    else:
        code = AIProviderErrorCode.UNKNOWN

    return AIProviderError(
        code,
        message,
        provider="openai",
        model=model,
        request_id=request_id,
        retryable=retryable,
        retry_after_seconds=retry_after,
        metadata={"status": status_code, "errorCategory": name},
    )


def _retry_after_seconds(exc: BaseException) -> int | None:
    response = getattr(exc, "response", None)
    headers = getattr(response, "headers", None)
    if headers is None:
        return None
    raw_value = headers.get("retry-after") or headers.get("Retry-After")
    if raw_value is None:
        return None
    try:
        value = int(str(raw_value).strip())
    except ValueError:
        return None
    return value if value >= 0 else None


def _error_text(exc: BaseException) -> str:
    parts = [str(exc)]
    for attr in ("body", "code", "type"):
        value = getattr(exc, attr, None)
        if value is not None:
            parts.append(str(value))
    return " ".join(parts)
