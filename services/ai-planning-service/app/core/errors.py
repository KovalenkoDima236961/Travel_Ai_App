import logging

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.privacy import redact_text

logger = logging.getLogger(__name__)


class ItineraryGenerationError(RuntimeError):
    """Raised when itinerary generation fails and no fallback is available."""


def register_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(ItineraryGenerationError)
    async def itinerary_generation_error_handler(
        request: Request, exc: ItineraryGenerationError
    ) -> JSONResponse:
        logger.warning(
            "Itinerary generation failed",
            extra={"path": request.url.path, "error": redact_text(str(exc), max_chars=500)},
        )
        return JSONResponse(
            status_code=500,
            content={"error": "Failed to generate itinerary"},
        )

    @app.exception_handler(Exception)
    async def unexpected_error_handler(request: Request, exc: Exception) -> JSONResponse:
        logger.error(
            "Unhandled request error",
            extra={"path": request.url.path, "errorType": type(exc).__name__},
        )
        return JSONResponse(
            status_code=500,
            content={"error": "Internal server error"},
        )
