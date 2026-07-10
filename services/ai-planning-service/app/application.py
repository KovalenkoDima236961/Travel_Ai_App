import logging
from dataclasses import dataclass

from fastapi import FastAPI

from app.api.destination_context_routes import router as destination_context_router
from app.api.knowledge_routes import router as knowledge_router
from app.api.routes import router
from app.config import Settings, get_settings
from app.core.errors import register_exception_handlers
from app.observability import metrics_response, request_context_middleware
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
from app.services.template_adapter import TemplateAdapter, get_template_adapter


@dataclass(frozen=True)
class ApplicationServices:
    itinerary_generator: ItineraryGenerator
    template_adapter: TemplateAdapter
    destination_knowledge_provider: DestinationKnowledgeProvider | None
    knowledge_search_service: KnowledgeSearchService | None
    destination_suggestion_generator: DestinationSuggestionGenerator


def create_app(settings: Settings | None = None) -> FastAPI:
    resolved_settings = settings or get_settings()
    logging.basicConfig(level=resolved_settings.log_level)
    services = build_application_services(resolved_settings)

    app = FastAPI(
        title="AI Planning Service",
        version="2.1.0",
        description="AI itinerary planner with mock and local Ollama generator modes.",
    )
    _configure_state(app, resolved_settings, services)
    _configure_observability(app)
    _configure_routes(app)
    return app


def build_application_services(settings: Settings) -> ApplicationServices:
    destination_knowledge_provider = get_destination_knowledge_provider(settings)
    knowledge_search_service = get_knowledge_search_service(settings)
    itinerary_generator = get_itinerary_generator(
        settings,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
    )
    template_adapter = get_template_adapter(settings)
    destination_suggestion_generator = get_destination_suggestion_generator(settings)

    return ApplicationServices(
        itinerary_generator=itinerary_generator,
        template_adapter=template_adapter,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
        destination_suggestion_generator=destination_suggestion_generator,
    )


def _configure_state(
    app: FastAPI,
    settings: Settings,
    services: ApplicationServices,
) -> None:
    app.state.settings = settings
    app.state.services = services
    app.state.itinerary_generator = services.itinerary_generator
    app.state.template_adapter = services.template_adapter
    app.state.destination_knowledge_provider = services.destination_knowledge_provider
    app.state.knowledge_search_service = services.knowledge_search_service
    app.state.destination_suggestion_generator = services.destination_suggestion_generator


def _configure_observability(app: FastAPI) -> None:
    register_exception_handlers(app)
    app.middleware("http")(request_context_middleware)
    app.add_api_route("/metrics", metrics_response, methods=["GET"], include_in_schema=False)


def _configure_routes(app: FastAPI) -> None:
    app.include_router(router)
    app.include_router(destination_context_router)
    app.include_router(knowledge_router)
