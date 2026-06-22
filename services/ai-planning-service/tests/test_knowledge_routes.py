from typing import Any

from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.itinerary_generator import MockItineraryGenerator


class FakeKnowledgeSearchService:
    def __init__(self, items: list[KnowledgeSearchResult]) -> None:
        self.items = items
        self.calls: list[dict[str, Any]] = []

    def search(
        self,
        destination: str,
        interests: list[str],
        query: str | None,
        top_k: int,
    ) -> list[KnowledgeSearchResult]:
        self.calls.append(
            {
                "destination": destination,
                "interests": interests,
                "query": query,
                "top_k": top_k,
            }
        )
        return self.items


def _client(settings: Settings, search_service: FakeKnowledgeSearchService | None) -> TestClient:
    app = create_app()
    app.state.settings = settings
    app.state.itinerary_generator = MockItineraryGenerator()
    app.state.knowledge_search_service = search_service
    return TestClient(app)


def test_knowledge_search_route_returns_results_from_search_service() -> None:
    search_service = FakeKnowledgeSearchService(
        [
            KnowledgeSearchResult(
                id="rome:food.md:0",
                destination="rome",
                source="food.md",
                content="Try carbonara.",
                score=0.82,
                metadata={"chunkIndex": 0},
            )
        ]
    )
    client = _client(Settings(rag_enabled=True), search_service)

    response = client.post(
        "/knowledge/search",
        json={
            "destination": "Rome",
            "interests": ["food", "hidden_gems"],
            "query": "local food",
            "topK": 5,
        },
    )

    assert response.status_code == 200
    assert response.json() == {
        "items": [
            {
                "id": "rome:food.md:0",
                "destination": "rome",
                "source": "food.md",
                "content": "Try carbonara.",
                "score": 0.82,
                "metadata": {"chunkIndex": 0},
            }
        ]
    }
    assert search_service.calls == [
        {
            "destination": "Rome",
            "interests": ["food", "hidden_gems"],
            "query": "local food",
            "top_k": 5,
        }
    ]


def test_knowledge_search_route_returns_empty_list_when_rag_disabled() -> None:
    search_service = FakeKnowledgeSearchService([])
    client = _client(Settings(rag_enabled=False), search_service)

    response = client.post(
        "/knowledge/search",
        json={"destination": "Rome", "interests": ["food"], "topK": 5},
    )

    assert response.status_code == 200
    assert response.json() == {"items": []}
    assert search_service.calls == []


def test_knowledge_search_route_rejects_invalid_top_k() -> None:
    client = _client(Settings(rag_enabled=True), FakeKnowledgeSearchService([]))

    response = client.post(
        "/knowledge/search",
        json={"destination": "Rome", "topK": 11},
    )

    assert response.status_code == 422
