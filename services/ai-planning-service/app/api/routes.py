import logging
import time
from typing import overload

from fastapi import APIRouter, Depends, Request
from fastapi.responses import JSONResponse
from pydantic import ValidationError

from app.api.dependencies import (
    get_configured_itinerary_generator,
    get_configured_settings,
    get_configured_template_adapter,
)
from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.core.readiness import check_readiness
from app.observability import record_ai_request, record_ai_validation_failure
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
from app.schemas.repair import RepairItineraryRequest, RepairItineraryResponse
from app.schemas.template_adaptation import TemplateAdaptationRequest, TemplateAdaptationResponse
from app.services.itinerary_generator import ItineraryGenerator
from app.services.template_adaptation_validator import TemplateAdaptationValidationError
from app.services.template_adapter import TemplateAdapter, validate_adaptation

logger = logging.getLogger(__name__)

router = APIRouter()


@router.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "ai-planning-service"}


@router.get("/ready")
def ready(settings: Settings = Depends(get_configured_settings)) -> JSONResponse:
    started_at = time.monotonic()
    result = check_readiness(settings)
    logger.info(
        "Readiness check completed",
        extra={
            "status": result.status,
            "checks": result.checks,
            "duration_ms": int((time.monotonic() - started_at) * 1000),
        },
    )
    return JSONResponse(
        status_code=result.status_code,
        content={"status": result.status, "checks": result.checks},
    )


@router.post("/generate-itinerary", response_model=ItineraryResponse)
def generate_itinerary(
    request: GenerateItineraryRequest,
    settings: Settings = Depends(get_configured_settings),
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> ItineraryResponse:
    operation = "generate_itinerary"
    mode = _generator_mode(settings)
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
    settings: Settings = Depends(get_configured_settings),
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateDayResponse | JSONResponse:
    operation = "regenerate_day"
    mode = _generator_mode(settings)
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
    settings: Settings = Depends(get_configured_settings),
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RegenerateItemResponse | JSONResponse:
    operation = "regenerate_item"
    mode = _generator_mode(settings)
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


@overload
async def _parse_partial_request(
    request: Request,
    model: type[RegenerateItemRequest],
    operation: str,
) -> RegenerateItemRequest | JSONResponse: ...


@overload
async def _parse_partial_request(
    request: Request,
    model: type[RegenerateDayRequest],
    operation: str,
) -> RegenerateDayRequest | JSONResponse: ...


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
    settings: Settings = Depends(get_configured_settings),
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> BudgetOptimizationProposalResponse:
    operation = "optimize_budget_day"
    mode = _generator_mode(settings)
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


@router.post("/repair-itinerary", response_model=RepairItineraryResponse)
def repair_itinerary(
    request: RepairItineraryRequest,
    settings: Settings = Depends(get_configured_settings),
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> RepairItineraryResponse:
    operation = "repair_itinerary"
    mode = _generator_mode(settings)
    started_at = time.monotonic()
    try:
        response = generator.repair_itinerary(request)
        record_ai_request(operation, "success", mode, time.monotonic() - started_at)
        return response
    except ItineraryGenerationError:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise
    except Exception as exc:
        record_ai_request(operation, "error", mode, time.monotonic() - started_at)
        raise ItineraryGenerationError("Failed to repair itinerary") from exc


@router.post("/adapt-template", response_model=TemplateAdaptationResponse)
def adapt_template(
    request: TemplateAdaptationRequest,
    settings: Settings = Depends(get_configured_settings),
    adapter: TemplateAdapter = Depends(get_configured_template_adapter),
) -> TemplateAdaptationResponse | JSONResponse:
    operation = "adapt_template"
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


def _generator_mode(settings: Settings) -> str:
    return settings.itinerary_generator_mode.strip().lower()


def _validation_error_message(exc: ValidationError) -> str:
    if not exc.errors():
        return "invalid request"
    first_error = exc.errors()[0]
    message = str(first_error.get("msg") or "invalid request")
    return message.removeprefix("Value error, ")
