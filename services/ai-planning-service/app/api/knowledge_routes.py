import logging

from fastapi import APIRouter, Depends

from app.api.dependencies import get_configured_knowledge_search_service, get_configured_settings
from app.config import Settings
from app.schemas.knowledge import KnowledgeSearchRequest, KnowledgeSearchResponse
from app.services.knowledge_search import KnowledgeSearchService

logger = logging.getLogger(__name__)

router = APIRouter()


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
