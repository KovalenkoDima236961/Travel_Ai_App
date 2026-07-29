import logging
from dataclasses import dataclass

from fastapi import FastAPI

from app.api.destination_context_routes import router as destination_context_router
from app.api.knowledge_routes import router as knowledge_router
from app.api.routes import router
from app.config import Settings, get_settings
from app.core.errors import register_exception_handlers
from app.observability import metrics_response, request_context_middleware
from app.providers.base import AIModelProvider
from app.providers.factory import build_openai_provider_if_needed
from app.services.copilot import CopilotResponder, get_copilot_responder
from app.services.destination_knowledge import DestinationKnowledgeProvider
from app.services.destination_suggestion import (
    DestinationSuggestionGenerator,
    get_destination_suggestion_generator,
)
from app.services.generator_factory import (
    get_destination_knowledge_provider,
    get_itinerary_generator,
    get_knowledge_search_service,
)
from app.services.itinerary_generator import ItineraryGenerator
from app.services.knowledge_search import KnowledgeSearchService
from app.services.route_alternatives import (
    RouteAlternativeGenerator,
    get_route_alternative_generator,
)
from app.services.template_adapter import TemplateAdapter, get_template_adapter
from app.services.trip_recap import TripRecapGenerator, get_trip_recap_generator
from app.version import get_version_info

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class ApplicationServices:
    ai_model_provider: AIModelProvider | None
    itinerary_generator: ItineraryGenerator
    template_adapter: TemplateAdapter
    destination_knowledge_provider: DestinationKnowledgeProvider | None
    knowledge_search_service: KnowledgeSearchService | None
    destination_suggestion_generator: DestinationSuggestionGenerator
    route_alternative_generator: RouteAlternativeGenerator
    copilot_responder: CopilotResponder
    trip_recap_generator: TripRecapGenerator


def create_app(settings: Settings | None = None) -> FastAPI:
    resolved_settings = settings or get_settings()
    logging.basicConfig(level=resolved_settings.log_level)
    if resolved_settings.allow_llm_payload_logging:
        logger.warning(
            "AI prompt logging enabled for local diagnostics; payloads are redacted and truncated"
        )
    else:
        logger.info("AI prompt logging disabled")
    _log_ai_provider_startup(resolved_settings)
    services = build_application_services(resolved_settings)

    app = FastAPI(
        title="AI Planning Service",
        version=get_version_info().version,
        description="AI itinerary planner with mock and local Ollama generator modes.",
    )
    _configure_state(app, resolved_settings, services)
    _configure_observability(app)
    _configure_routes(app)
    return app


def build_application_services(settings: Settings) -> ApplicationServices:
    destination_knowledge_provider = get_destination_knowledge_provider(settings)
    knowledge_search_service = get_knowledge_search_service(settings)
    ai_model_provider = build_openai_provider_if_needed(
        settings,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
    )
    itinerary_generator = get_itinerary_generator(
        settings,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
        openai_provider=ai_model_provider,
    )
    template_adapter = get_template_adapter(settings)
    destination_suggestion_generator = get_destination_suggestion_generator(
        settings, openai_provider=ai_model_provider
    )
    route_alternative_generator = get_route_alternative_generator(
        settings, openai_provider=ai_model_provider
    )
    copilot_responder = get_copilot_responder(settings, openai_provider=ai_model_provider)
    trip_recap_generator = get_trip_recap_generator(settings, openai_provider=ai_model_provider)

    return ApplicationServices(
        ai_model_provider=ai_model_provider,
        itinerary_generator=itinerary_generator,
        template_adapter=template_adapter,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
        destination_suggestion_generator=destination_suggestion_generator,
        route_alternative_generator=route_alternative_generator,
        copilot_responder=copilot_responder,
        trip_recap_generator=trip_recap_generator,
    )


def _configure_state(
    app: FastAPI,
    settings: Settings,
    services: ApplicationServices,
) -> None:
    app.state.settings = settings
    app.state.services = services
    app.state.ai_model_provider = services.ai_model_provider
    app.state.itinerary_generator = services.itinerary_generator
    app.state.template_adapter = services.template_adapter
    app.state.destination_knowledge_provider = services.destination_knowledge_provider
    app.state.knowledge_search_service = services.knowledge_search_service
    app.state.destination_suggestion_generator = services.destination_suggestion_generator
    app.state.route_alternative_generator = services.route_alternative_generator
    app.state.copilot_responder = services.copilot_responder
    app.state.trip_recap_generator = services.trip_recap_generator


def _configure_observability(app: FastAPI) -> None:
    register_exception_handlers(app)
    app.middleware("http")(request_context_middleware)
    app.add_api_route("/metrics", metrics_response, methods=["GET"], include_in_schema=False)

    @app.on_event("shutdown")
    async def close_ai_model_provider() -> None:
        provider = getattr(app.state, "ai_model_provider", None)
        close = getattr(provider, "close", None)
        if close is not None:
            await close()


def _configure_routes(app: FastAPI) -> None:
    app.include_router(router)
    app.include_router(destination_context_router)
    app.include_router(knowledge_router)


def _log_ai_provider_startup(settings: Settings) -> None:
    safe_models = {
        "default": bool(settings.openai_model_default.strip()),
        "itinerary": bool(settings.openai_model_itinerary.strip()),
        "regeneration": bool(settings.openai_model_regeneration.strip()),
        "repair": bool(settings.openai_model_repair.strip()),
        "discovery": bool(settings.openai_model_discovery.strip()),
        "routeAlternatives": bool(settings.openai_model_route_alternatives.strip()),
        "budgetOptimization": bool(settings.openai_model_budget_optimization.strip()),
        "checklist": bool(settings.openai_model_checklist.strip()),
        "copilot": bool(settings.openai_model_copilot.strip()),
        "recap": bool(settings.openai_model_recap.strip()),
        "evaluation": bool(settings.openai_model_evaluation.strip()),
    }
    logger.info(
        "AI provider startup configuration",
        extra={
            "activeProvider": settings.ai_model_provider.strip().lower(),
            "itineraryMode": settings.itinerary_generator_mode.strip().lower(),
            "copilotMode": settings.copilot_mode.strip().lower(),
            "tripRecapMode": settings.trip_recap_mode.strip().lower(),
            "fallbackProvider": settings.ai_model_provider_fallback.strip().lower(),
            "openaiEnabled": settings.openai_enabled,
            "openaiConfiguredModelAliases": safe_models,
            "openaiTimeoutSeconds": settings.openai_timeout_seconds,
            "openaiMaxRetries": settings.openai_max_retries,
            "openaiStoreResponses": settings.openai_store_responses,
            "openaiUsageTrackingEnabled": settings.openai_usage_tracking_enabled,
        },
    )
