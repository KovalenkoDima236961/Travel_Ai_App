from fastapi import Request

from app.config import Settings
from app.services.destination_knowledge import DestinationKnowledgeProvider
from app.services.destination_suggestion import DestinationSuggestionGenerator
from app.services.itinerary_generator import ItineraryGenerator
from app.services.knowledge_search import KnowledgeSearchService
from app.services.template_adapter import TemplateAdapter


def get_configured_settings(request: Request) -> Settings:
    return request.app.state.settings


def get_configured_itinerary_generator(request: Request) -> ItineraryGenerator:
    return request.app.state.itinerary_generator


def get_configured_template_adapter(request: Request) -> TemplateAdapter:
    return request.app.state.template_adapter


def get_configured_destination_knowledge_provider(
    request: Request,
) -> DestinationKnowledgeProvider | None:
    return getattr(request.app.state, "destination_knowledge_provider", None)


def get_configured_knowledge_search_service(request: Request) -> KnowledgeSearchService | None:
    return getattr(request.app.state, "knowledge_search_service", None)


def get_configured_destination_suggestion_generator(
    request: Request,
) -> DestinationSuggestionGenerator:
    return request.app.state.destination_suggestion_generator
