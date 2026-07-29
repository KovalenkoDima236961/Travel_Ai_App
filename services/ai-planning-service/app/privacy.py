"""Privacy and prompt-injection guards shared by AI request paths."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

REDACTED = "[REDACTED]"

_EMAIL = re.compile(r"\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b", re.IGNORECASE)
_PHONE = re.compile(r"\+?[0-9][0-9 ()\-.]{7,}[0-9]")
_UUID = re.compile(
    r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b",
    re.IGNORECASE,
)
_BEARER = re.compile(r"\bbearer\s+[a-z0-9._~+/=\-]{12,}", re.IGNORECASE)
_SECRET = re.compile(
    r"\b(?:sk|pk|api[_-]?key|token|secret)[_:\-= ]+[a-z0-9_./+\-=]{12,}",
    re.IGNORECASE,
)

_INJECTION_PATTERNS = (
    re.compile(r"ignore\s+(?:all\s+)?previous\s+instructions", re.IGNORECASE),
    re.compile(r"\bsystem\s+prompt\b", re.IGNORECASE),
    re.compile(r"\bdeveloper\s+message\b", re.IGNORECASE),
    re.compile(r"\bexfiltrat(?:e|ion)\b", re.IGNORECASE),
    re.compile(r"\bapi[_ -]?key\b", re.IGNORECASE),
)

SANITIZER_VERSION = "ai_privacy_sanitizer_v1"

_SENSITIVE_KEYWORDS = {
    "access_token",
    "accesstoken",
    "authorization",
    "auth_header",
    "calendar_description",
    "calendar_event_description",
    "calendar_event_title",
    "calendar_title",
    "comment",
    "comments",
    "cookie",
    "cookies",
    "email",
    "home_address",
    "internal_log",
    "log",
    "oauth_token",
    "ocr_text",
    "phone",
    "private_note",
    "private_notes",
    "raw_provider_payload",
    "raw_response",
    "receipt",
    "receipt_data",
    "refresh_token",
    "share_password",
    "share_token",
    "stack_trace",
    "token",
    "user_id",
    "userid",
}


@dataclass(frozen=True)
class UntrustedContent:
    content: str
    suspicious: bool
    warning_codes: tuple[str, ...]


@dataclass(frozen=True)
class AISanitizationResult:
    sanitized_payload: Any
    removed_fields: tuple[str, ...]
    warnings: tuple[str, ...]
    blocked: bool
    sanitizer_version: str = SANITIZER_VERSION


def redact_text(value: str, max_chars: int | None = None) -> str:
    redacted = value
    for pattern in (_EMAIL, _UUID, _BEARER, _SECRET):
        redacted = pattern.sub(REDACTED, redacted)
    redacted = _PHONE.sub(
        lambda match: (
            REDACTED
            if sum(character.isdigit() for character in match.group(0)) >= 10
            else match.group(0)
        ),
        redacted,
    )
    if max_chars is not None and len(redacted) > max_chars:
        redacted = redacted[:max_chars] + "…[truncated]"
    return redacted


def guard_untrusted_content(value: str, max_chars: int = 2_000) -> UntrustedContent:
    cleaned = redact_text(value.strip(), max_chars=max_chars)
    warnings: list[str] = []
    for index, pattern in enumerate(_INJECTION_PATTERNS):
        if pattern.search(cleaned):
            warnings.append(f"prompt_injection_pattern_{index + 1}")

    # Suspicious document instructions are neutralized but the travel facts are
    # retained. The surrounding prompt also labels the entire block untrusted.
    if warnings:
        for pattern in _INJECTION_PATTERNS:
            cleaned = pattern.sub("[UNTRUSTED_INSTRUCTION_REMOVED]", cleaned)

    return UntrustedContent(
        content=cleaned,
        suspicious=bool(warnings),
        warning_codes=tuple(warnings),
    )


def sanitize_ai_payload(value: Any, max_string_chars: int = 6_000) -> AISanitizationResult:
    """Sanitize model-bound payloads without retaining removed values."""
    removed: list[str] = []
    warnings: list[str] = []

    def sanitize(current: Any, path: str) -> Any:
        if isinstance(current, str):
            guarded = guard_untrusted_content(current, max_chars=max_string_chars)
            warnings.extend(f"{path}:{code}" for code in guarded.warning_codes)
            return guarded.content
        if isinstance(current, list):
            return [sanitize(item, f"{path}[]") for item in current]
        if isinstance(current, tuple):
            return tuple(sanitize(item, f"{path}[]") for item in current)
        if isinstance(current, dict):
            sanitized: dict[Any, Any] = {}
            for key, item in current.items():
                key_text = str(key)
                next_path = f"{path}.{key_text}" if path else key_text
                if _is_sensitive_key(key_text):
                    removed.append(next_path)
                    continue
                sanitized[key] = sanitize(item, next_path)
            return sanitized
        return current

    sanitized_payload = sanitize(value, "")
    return AISanitizationResult(
        sanitized_payload=sanitized_payload,
        removed_fields=tuple(removed),
        warnings=tuple(dict.fromkeys(warnings)),
        blocked=False,
    )


def _is_sensitive_key(key: str) -> bool:
    normalized = re.sub(r"[^a-z0-9]+", "_", key.strip().lower()).strip("_")
    if normalized in _SENSITIVE_KEYWORDS:
        return True
    return any(part in _SENSITIVE_KEYWORDS for part in normalized.split("_"))
