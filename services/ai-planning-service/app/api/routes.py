import logging
import time
from pathlib import Path

import httpx
from fastapi import APIRouter, Depends, Request
from fastapi.responses import JSONResponse
from pydantic import ValidationError

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.observability import (
    record_ai_request,
    record_ai_validation_failure,
)
from app.schemas.itinerary import (
    BudgetOptimizationProposalResponse,
    GenerateItineraryRequest,
    ItineraryResponse,
    OptimizeBudgetDayRequest,
    RegenerateDayRequest,
    RegenerateDayResponse,
    RegenerateItemRequest,
    RegenerateItemResponse,
)
from app.schemas.template_adaptation import (
    TemplateAdaptationRequest,
    TemplateAdaptationResponse,
)
from app.services.chroma_client import create_persistent_chroma_client
from app.services.itinerary_generator import ItineraryGenerator
from app.services.template_adaptation_validator import TemplateAdaptationValidationError
from app.services.template_adapter import TemplateAdapter, validate_adaptation

logger = logging.getLogger(__name__)

router = APIRouter()


def get_configured_itinerary_generator(request: Request) -> ItineraryGenerator:
    return request.app.state.itinerary_generator


def get_configured_template_adapter(request: Request) -> TemplateAdapter:
    return request.app.state.template_adapter


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
    http_request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> ItineraryResponse:
    operation = "generate_itinerary"
    mode = _generator_mode(http_request)
    started_at = time.monotonic()
    try:
        response = generator.generate(request)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to generate itinerary") from exc


@router.post("/regenerate-day", response_model=RegenerateDayResponse)
async def regenerate_day(
    request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateDayResponse | JSONResponse:
    operation = "regenerate_day"
    mode = _generator_mode(request)
    started_at = time.monotonic()
    parsed = await _parse_partial_request(request, RegenerateDayRequest, operation)
    if isinstance(parsed, JSONResponse):
        record_ai_request(operation, "validation_error", mode, time.monotonic() - started_at)
        return parsed

    try:
        response = generator.regenerate_day(parsed)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to regenerate itinerary day") from exc


@router.post("/regenerate-item", response_model=RegenerateItemResponse)
async def regenerate_item(
    request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateItemResponse | JSONResponse:
    operation = "regenerate_item"
    mode = _generator_mode(request)
    started_at = time.monotonic()
    parsed = await _parse_partial_request(request, RegenerateItemRequest, operation)
    if isinstance(parsed, JSONResponse):
        record_ai_request(operation, "validation_error", mode, time.monotonic() - started_at)
        return parsed

    try:
        response = generator.regenerate_item(parsed)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to regenerate itinerary item") from exc


async def _parse_partial_request(
    request: Request,
    model: type[RegenerateDayRequest] | type[RegenerateItemRequest],
    operation: str,
) -> RegenerateDayRequest | RegenerateItemRequest | JSONResponse:
    try:
        payload = await request.json()
    except ValueError:
        record_ai_validation_failure(operation)
        return JSONResponse(status_code=400, content={"error": "invalid request body"})

    try:
        return model.model_validate(payload)
    except ValidationError as exc:
        record_ai_validation_failure(operation)
        return JSONResponse(
            status_code=400,
            content={"error": _validation_error_message(exc)},
        )


@router.post("/optimize-budget/day", response_model=BudgetOptimizationProposalResponse)
def optimize_budget_day(
    request: OptimizeBudgetDayRequest,
    http_request: Request,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> BudgetOptimizationProposalResponse:
    operation = "optimize_budget_day"
    mode = _generator_mode(http_request)
    started_at = time.monotonic()
    try:
        response = generator.optimize_budget_day(request)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to optimize itinerary budget") from exc


@router.post("/adapt-template", response_model=TemplateAdaptationResponse)
def adapt_template(
    request: TemplateAdaptationRequest,
    http_request: Request,
    adapter: TemplateAdapter = Depends(get_configured_template_adapter),
) -> TemplateAdaptationResponse | JSONResponse:
    operation = "adapt_template"
    settings: Settings = http_request.app.state.settings
    if not settings.template_adaptation_enabled:
        return JSONResponse(
            status_code=503,
            content={"error": "template adaptation is disabled"},
        )
    mode = settings.template_adaptation_mode.strip().lower()
    started_at = time.monotonic()
    try:
        response = adapter.adapt(request)
        # Attach validation warnings (mock output is not validated internally) and
        # reject structurally-invalid output so a broken adaptation never returns.
        validate_adaptation(request, response)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except TemplateAdaptationValidationError as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        record_ai_validation_failure(operation)
        raise ItineraryGenerationError("Failed to adapt template") from exc
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to adapt template") from exc


def _generator_mode(request: Request) -> str:
    settings: Settings = request.app.state.settings
    return settings.itinerary_generator_mode.strip().lower()


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
