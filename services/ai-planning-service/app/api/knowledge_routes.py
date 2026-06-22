import logging

from fastapi import APIRouter, Depends, Request

from app.config import Settings
from app.schemas.knowledge import KnowledgeSearchRequest, KnowledgeSearchResponse
from app.services.knowledge_search import KnowledgeSearchService

logger = logging.getLogger(__name__)

router = APIRouter()


def get_configured_settings(request: Request) -> Settings:
    return request.app.state.settings


def get_configured_knowledge_search_service(request: Request) -> KnowledgeSearchService | None:
    return getattr(request.app.state, "knowledge_search_service", None)


@router.post("/knowledge/search", response_model=KnowledgeSearchResponse)
def search_knowledge(
    search_request: KnowledgeSearchRequest,
    settings: Settings = Depends(get_configured_settings),
    search_service: KnowledgeSearchService | None = Depends(
        get_configured_knowledge_search_service
    ),
) -> KnowledgeSearchResponse:
    logger.info(
        "Knowledge search requested",
        extra={
            "destination": search_request.destination,
            "rag_enabled": settings.rag_enabled,
            "top_k": search_request.topK,
        },
    )

    if not settings.rag_enabled or search_service is None:
        return KnowledgeSearchResponse(items=[])

    items = search_service.search(
        destination=search_request.destination,
        interests=search_request.interests,
        query=search_request.query,
        top_k=search_request.topK,
    )
    return KnowledgeSearchResponse(items=items)
