from fastapi import APIRouter, Depends, Request

from app.core.errors import ItineraryGenerationError
from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.services.itinerary_generator import ItineraryGenerator

router = APIRouter()


def get_configured_itinerary_generator(request: Request) -> ItineraryGenerator:
    return request.app.state.itinerary_generator


@router.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "ai-planning-service"}


@router.post("/generate-itinerary", response_model=ItineraryResponse)
def generate_itinerary(
    request: GenerateItineraryRequest,
    generator: ItineraryGenerator = Depends(get_configured_itinerary_generator),
) -> ItineraryResponse:
    try:
        return generator.generate(request)
    except ItineraryGenerationError:
        raise
    except Exception as exc:
        raise ItineraryGenerationError("Failed to generate itinerary") from exc
