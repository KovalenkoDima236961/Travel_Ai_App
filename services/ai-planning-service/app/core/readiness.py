import logging
from dataclasses import dataclass

import httpx

from app.config import Settings
from app.core.paths import resolve_service_path
from app.services.chroma_client import create_persistent_chroma_client

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class ReadinessResult:
    is_ready: bool
    checks: dict[str, str]

    @property
    def status(self) -> str:
        return "ready" if self.is_ready else "not_ready"

    @property
    def status_code(self) -> int:
        return 200 if self.is_ready else 503


def check_readiness(settings: Settings) -> ReadinessResult:
    checks: dict[str, str] = {"app": "ok"}
    is_ready = True

    if settings.itinerary_generator_mode.strip().lower() == "ollama":
        ollama_error = _check_ollama(settings)
        if ollama_error is None:
            checks["ollama"] = "ok"
        else:
            checks["ollama"] = "failed"
            is_ready = False
            logger.warning(
                "Ollama readiness check failed",
                extra={"ollama_base_url": settings.ollama_base_url},
                exc_info=ollama_error,
            )

    if settings.rag_enabled:
        chroma_error = _check_chroma(settings)
        if chroma_error is None:
            checks["chroma"] = "ok"
        else:
            checks["chroma"] = "failed"
            is_ready = False
            logger.warning(
                "ChromaDB readiness check failed",
                extra={"rag_chroma_dir": settings.rag_chroma_dir},
                exc_info=chroma_error,
            )

    return ReadinessResult(is_ready=is_ready, checks=checks)


def _check_ollama(settings: Settings) -> BaseException | None:
    endpoint = f"{settings.ollama_base_url.rstrip('/')}/api/tags"
    timeout = min(settings.ollama_timeout_seconds, 5)
    try:
        with httpx.Client(timeout=timeout) as client:
            response = client.get(endpoint)
            response.raise_for_status()
    except Exception as exc:
        return exc
    return None


def _check_chroma(settings: Settings) -> BaseException | None:
    try:
        chroma_dir = resolve_service_path(settings.rag_chroma_dir)
        chroma_dir.mkdir(parents=True, exist_ok=True)
        create_persistent_chroma_client(settings, chroma_dir)
    except Exception as exc:
        return exc
    return None
