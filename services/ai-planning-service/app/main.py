import logging

from fastapi import FastAPI

from app.api.destination_context_routes import router as destination_context_router
from app.api.routes import router
from app.config import get_settings
from app.core.errors import register_exception_handlers
from app.services.generator_factory import (
    get_destination_knowledge_provider,
    get_itinerary_generator,
)


def create_app() -> FastAPI:
    settings = get_settings()
    logging.basicConfig(level=settings.log_level)
    destination_knowledge_provider = get_destination_knowledge_provider(settings)
    generator = get_itinerary_generator(
        settings,
        destination_knowledge_provider=destination_knowledge_provider,
    )

    app = FastAPI(
        title="AI Planning Service",
        version="2.1.0",
        description="AI itinerary planner with mock and local Ollama generator modes.",
    )
    app.state.settings = settings
    app.state.itinerary_generator = generator
    app.state.destination_knowledge_provider = destination_knowledge_provider
    register_exception_handlers(app)
    app.include_router(router)
    app.include_router(destination_context_router)
    return app


app = create_app()
