import logging

from fastapi import FastAPI

from app.api.routes import router
from app.config import get_settings
from app.core.errors import register_exception_handlers


def create_app() -> FastAPI:
    settings = get_settings()
    logging.basicConfig(level=settings.log_level)

    app = FastAPI(
        title="AI Planning Service",
        version="1.0.0",
        description="Deterministic v1 mock itinerary planner for service-to-service integration.",
    )
    register_exception_handlers(app)
    app.include_router(router)
    return app


app = create_app()
