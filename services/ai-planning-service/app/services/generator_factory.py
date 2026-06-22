import logging
from pathlib import Path

from app.config import Settings
from app.services.destination_knowledge import (
    DestinationKnowledgeProvider,
    FileDestinationKnowledgeProvider,
)
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.knowledge_search import KnowledgeSearchService
from app.services.ollama_itinerary_generator import OllamaItineraryGenerator

logger = logging.getLogger(__name__)


def get_itinerary_generator(
    settings: Settings,
    destination_knowledge_provider: DestinationKnowledgeProvider | None = None,
    knowledge_search_service: KnowledgeSearchService | None = None,
) -> ItineraryGenerator:
    mode = settings.itinerary_generator_mode.strip().lower() or "mock"

    if mode == "mock":
        return MockItineraryGenerator()

    if mode == "ollama":
        if not settings.ollama_base_url.strip():
            raise ValueError("OLLAMA_BASE_URL is required when ITINERARY_GENERATOR_MODE=ollama")
        if not settings.ollama_model.strip():
            raise ValueError("OLLAMA_MODEL is required when ITINERARY_GENERATOR_MODE=ollama")
        return OllamaItineraryGenerator(
            settings=settings,
            destination_knowledge_provider=(
                destination_knowledge_provider or get_destination_knowledge_provider(settings)
            ),
            knowledge_search_service=knowledge_search_service
            or get_knowledge_search_service(settings),
        )

    raise ValueError(
        "Unknown ITINERARY_GENERATOR_MODE "
        f"{settings.itinerary_generator_mode!r}; expected 'mock' or 'ollama'"
    )


def get_destination_knowledge_provider(settings: Settings) -> DestinationKnowledgeProvider | None:
    if not settings.destination_context_enabled:
        return None

    data_dir = _resolve_destination_context_dir(settings.destination_context_dir)
    if not data_dir.exists() or not data_dir.is_dir():
        logger.warning(
            "Destination context directory is missing or invalid",
            extra={"destination_context_dir": str(data_dir)},
        )
        return None

    return FileDestinationKnowledgeProvider(data_dir=data_dir)


def get_knowledge_search_service(settings: Settings) -> KnowledgeSearchService | None:
    if not settings.rag_enabled:
        return None
    return KnowledgeSearchService(settings=settings)


def _resolve_destination_context_dir(raw_data_dir: str) -> Path:
    data_dir = Path(raw_data_dir)
    if data_dir.is_absolute():
        return data_dir

    cwd_data_dir = Path.cwd() / data_dir
    if cwd_data_dir.exists():
        return cwd_data_dir

    service_root = Path(__file__).resolve().parents[2]
    return service_root / data_dir
