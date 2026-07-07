import logging

from fastapi import APIRouter, Depends
from fastapi.responses import JSONResponse

from app.api.dependencies import (
    get_configured_destination_knowledge_provider,
    get_configured_settings,
)
from app.config import Settings
from app.schemas.destination_context import (
    DestinationContext,
    DestinationContextListResponse,
    DestinationContextNotFoundResponse,
    DestinationContextPromptPreviewResponse,
)
from app.schemas.itinerary import GenerateItineraryRequest
from app.services.destination_knowledge import DestinationKnowledgeProvider
from app.services.prompt_builder import build_itinerary_prompt

logger = logging.getLogger(__name__)

router = APIRouter()


@router.get("/destination-context", response_model=DestinationContextListResponse)
def list_destination_contexts(
    settings: Settings = Depends(get_configured_settings),
    provider: DestinationKnowledgeProvider | None = Depends(
        get_configured_destination_knowledge_provider
    ),
) -> DestinationContextListResponse:
    logger.info(
        "Destination context list requested",
        extra={"destination_context_enabled": settings.destination_context_enabled},
    )

    if not settings.destination_context_enabled or provider is None:
        return DestinationContextListResponse(items=[])

    try:
        return DestinationContextListResponse(items=provider.list_contexts())
    except Exception:
        logger.warning("Destination context list failed", exc_info=True)
        return DestinationContextListResponse(items=[])


@router.get(
    "/destination-context/{destination}",
    response_model=DestinationContext,
    responses={404: {"model": DestinationContextNotFoundResponse}},
)
def get_destination_context(
    destination: str,
    settings: Settings = Depends(get_configured_settings),
    provider: DestinationKnowledgeProvider | None = Depends(
        get_configured_destination_knowledge_provider
    ),
) -> DestinationContext | JSONResponse:
    logger.info(
        "Destination context lookup requested",
        extra={
            "destination": destination,
            "destination_context_enabled": settings.destination_context_enabled,
        },
    )
    context = _lookup_destination_context(destination, settings, provider)
    logger.info(
        "Destination context lookup completed",
        extra={"destination": destination, "destination_context_found": context is not None},
    )

    if context is None:
        return JSONResponse(
            status_code=404,
            content=DestinationContextNotFoundResponse(
                error="Destination context not found"
            ).model_dump(),
        )

    return context


@router.post(
    "/destination-context/{destination}/preview-prompt",
    response_model=DestinationContextPromptPreviewResponse,
)
def preview_destination_context_prompt(
    destination: str,
    itinerary_request: GenerateItineraryRequest,
    settings: Settings = Depends(get_configured_settings),
    provider: DestinationKnowledgeProvider | None = Depends(
        get_configured_destination_knowledge_provider
    ),
) -> DestinationContextPromptPreviewResponse:
    logger.info(
        "Destination context prompt preview requested",
        extra={
            "destination": destination,
            "request_destination": itinerary_request.destination,
            "destination_context_enabled": settings.destination_context_enabled,
        },
    )
    context = _lookup_destination_context(destination, settings, provider)
    prompt = build_itinerary_prompt(itinerary_request, destination_context=context)

    log_context = {
        "destination": destination,
        "request_destination": itinerary_request.destination,
        "destination_context_found": context is not None,
    }
    logger.info("Destination context prompt preview completed", extra=log_context)
    if settings.allow_llm_payload_logging:
        logger.info(
            "Destination context prompt preview payload", extra={**log_context, "prompt": prompt}
        )

    return DestinationContextPromptPreviewResponse(
        destination_context_found=context is not None,
        destination_context=context,
        prompt=prompt,
    )


def _lookup_destination_context(
    destination: str,
    settings: Settings,
    provider: DestinationKnowledgeProvider | None,
) -> DestinationContext | None:
    if not settings.destination_context_enabled or provider is None:
        return None

    try:
        return provider.get_context(destination)
    except Exception:
        logger.warning(
            "Destination context lookup failed; continuing without destination context",
            extra={"destination": destination},
            exc_info=True,
        )
        return None
