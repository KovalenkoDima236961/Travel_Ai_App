import logging
import time
from pathlib import Path

import httpx
from fastapi import APIRouter, Depends, Request
from fastapi.responses import JSONResponse
from pydantic import ValidationError

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.itinerary import (
    GenerateItineraryRequest,
    ItineraryResponse,
    RegenerateDayRequest,
    RegenerateDayResponse,
    RegenerateItemRequest,
    RegenerateItemResponse,
)
from app.services.chroma_client import create_persistent_chroma_client
from app.services.itinerary_generator import ItineraryGenerator

logger = logging.getLogger(__name__)

router = APIRouter()


def get_configured_itinerary_generator(request: Request) -> ItineraryGenerator:
    return request.app.state.itinerary_generator


@router.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "ai-planning-service"}


@router.get("/ready")
def ready(request: Request) -> JSONResponse:
    started_at = time.monotonic()
    settings: Settings = request.app.state.settings
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

    status = "ready" if is_ready else "not_ready"
    status_code = 200 if is_ready else 503
    logger.info(
        "Readiness check completed",
        extra={
            "status": status,
            "checks": checks,
            "duration_ms": int((time.monotonic() - started_at) * 1000),
        },
    )
    return JSONResponse(status_code=status_code, content={"status": status, "checks": checks})


@router.post("/generate-itinerary", response_model=ItineraryResponse)
def generate_itinerary(
    request: GenerateItineraryRequest,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> ItineraryResponse:
    try:
        return generator.generate(request)
    except ItineraryGenerationError:
        raise
    except Exception as exc:
        raise ItineraryGenerationError("Failed to generate itinerary") from exc


@router.post("/regenerate-day", response_model=RegenerateDayResponse)
async def regenerate_day(
    request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateDayResponse | JSONResponse:
    parsed = await _parse_partial_request(request, RegenerateDayRequest)
    if isinstance(parsed, JSONResponse):
        return parsed

    try:
        return generator.regenerate_day(parsed)
    except ItineraryGenerationError:
        raise
    except Exception as exc:
        raise ItineraryGenerationError("Failed to regenerate itinerary day") from exc


@router.post("/regenerate-item", response_model=RegenerateItemResponse)
async def regenerate_item(
    request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateItemResponse | JSONResponse:
    parsed = await _parse_partial_request(request, RegenerateItemRequest)
    if isinstance(parsed, JSONResponse):
        return parsed

    try:
        return generator.regenerate_item(parsed)
    except ItineraryGenerationError:
        raise
    except Exception as exc:
        raise ItineraryGenerationError("Failed to regenerate itinerary item") from exc


async def _parse_partial_request(
    request: Request,
    model: type[RegenerateDayRequest] | type[RegenerateItemRequest],
) -> RegenerateDayRequest | RegenerateItemRequest | JSONResponse:
    try:
        payload = await request.json()
    except ValueError:
        return JSONResponse(status_code=400, content={"error": "invalid request body"})

    try:
        return model.model_validate(payload)
    except ValidationError as exc:
        return JSONResponse(
            status_code=400,
            content={"error": _validation_error_message(exc)},
        )


def _validation_error_message(exc: ValidationError) -> str:
    if not exc.errors():
        return "invalid request"
    first_error = exc.errors()[0]
    message = str(first_error.get("msg") or "invalid request")
    return message.removeprefix("Value error, ")


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
        chroma_dir = _resolve_service_path(settings.rag_chroma_dir)
        chroma_dir.mkdir(parents=True, exist_ok=True)

        create_persistent_chroma_client(settings, chroma_dir)
    except Exception as exc:
        return exc
    return None


def _resolve_service_path(raw_path: str) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    cwd_path = Path.cwd() / path
    if cwd_path.exists():
        return cwd_path

    service_root = Path(__file__).resolve().parents[2]
    return service_root / path
