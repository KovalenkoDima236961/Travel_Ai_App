import logging

from fastapi import FastAPI

from app.api.destination_context_routes import router as destination_context_router
from app.api.knowledge_routes import router as knowledge_router
from app.api.routes import router
from app.config import get_settings
from app.core.errors import register_exception_handlers
from app.observability import metrics_response, request_context_middleware
from app.services.generator_factory import (
    get_destination_knowledge_provider,
    get_itinerary_generator,
    get_knowledge_search_service,
)
from app.services.template_adapter import get_template_adapter


def create_app() -> FastAPI:
    settings = get_settings()
    logging.basicConfig(level=settings.log_level)
    destination_knowledge_provider = get_destination_knowledge_provider(settings)
    knowledge_search_service = get_knowledge_search_service(settings)
    generator = get_itinerary_generator(
        settings,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
    )
    template_adapter = get_template_adapter(settings)

    app = FastAPI(
        title="AI Planning Service",
        version="2.1.0",
        description="AI itinerary planner with mock and local Ollama generator modes.",
    )
    app.state.settings = settings
    app.state.itinerary_generator = generator
    app.state.template_adapter = template_adapter
    app.state.destination_knowledge_provider = destination_knowledge_provider
    app.state.knowledge_search_service = knowledge_search_service
    register_exception_handlers(app)
    app.middleware("http")(request_context_middleware)
    app.add_api_route("/metrics", metrics_response, methods=["GET"], include_in_schema=False)
    app.include_router(router)
    app.include_router(destination_context_router)
    app.include_router(knowledge_router)
    return app


app = create_app()
