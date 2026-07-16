"""Privacy and prompt-injection guards shared by AI request paths."""

from __future__ import annotations

import re
from dataclasses import dataclass

REDACTED = "[REDACTED]"

_EMAIL = re.compile(r"\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b", re.IGNORECASE)
_PHONE = re.compile(r"\+?[0-9][0-9 ()\-.]{7,}[0-9]")
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


@dataclass(frozen=True)
class UntrustedContent:
    content: str
    suspicious: bool
    warning_codes: tuple[str, ...]


def redact_text(value: str, max_chars: int | None = None) -> str:
    redacted = value
    for pattern in (_EMAIL, _BEARER, _SECRET):
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
