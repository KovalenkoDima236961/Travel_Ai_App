from fastapi import APIRouter, Depends

from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.services.itinerary_generator import MockItineraryGenerator

router = APIRouter()

_generator = MockItineraryGenerator()


def get_itinerary_generator() -> MockItineraryGenerator:
    return _generator


@router.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "ai-planning-service"}


@router.post("/generate-itinerary", response_model=ItineraryResponse)
def generate_itinerary(
    request: GenerateItineraryRequest,
    generator: MockItineraryGenerator = Depends(get_itinerary_generator),
) -> ItineraryResponse:
    return generator.generate(request)
