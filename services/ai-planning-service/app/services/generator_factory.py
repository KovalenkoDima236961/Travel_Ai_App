from app.config import Settings
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.ollama_itinerary_generator import OllamaItineraryGenerator


def get_itinerary_generator(settings: Settings) -> ItineraryGenerator:
    mode = settings.itinerary_generator_mode.strip().lower() or "mock"

    if mode == "mock":
        return MockItineraryGenerator()

    if mode == "ollama":
        if not settings.ollama_base_url.strip():
            raise ValueError("OLLAMA_BASE_URL is required when ITINERARY_GENERATOR_MODE=ollama")
        if not settings.ollama_model.strip():
            raise ValueError("OLLAMA_MODEL is required when ITINERARY_GENERATOR_MODE=ollama")
        return OllamaItineraryGenerator(settings=settings)

    raise ValueError(
        "Unknown ITINERARY_GENERATOR_MODE "
        f"{settings.itinerary_generator_mode!r}; expected 'mock' or 'ollama'"
    )
