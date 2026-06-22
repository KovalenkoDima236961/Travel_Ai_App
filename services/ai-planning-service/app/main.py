import logging

from fastapi import FastAPI

from app.api.routes import router
from app.config import get_settings
from app.core.errors import register_exception_handlers
from app.services.generator_factory import get_itinerary_generator


def create_app() -> FastAPI:
    settings = get_settings()
    logging.basicConfig(level=settings.log_level)
    generator = get_itinerary_generator(settings)

    app = FastAPI(
        title="AI Planning Service",
        version="2.1.0",
        description="AI itinerary planner with mock and local Ollama generator modes.",
    )
    app.state.settings = settings
    app.state.itinerary_generator = generator
    register_exception_handlers(app)
    app.include_router(router)
    return app


app = create_app()
