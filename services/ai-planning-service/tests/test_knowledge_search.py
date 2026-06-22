from typing import Any

from app.config import Settings
from app.services.knowledge_search import KnowledgeSearchService


class FakeEmbeddingClient:
    def __init__(self, *, fail: bool = False) -> None:
        self.fail = fail
        self.texts: list[str] = []

    def embed(self, text: str) -> list[float]:
        self.texts.append(text)
        if self.fail:
            raise RuntimeError("embedding failed")
        return [0.1, 0.2, 0.3]


class FakeCollection:
    def __init__(self, result: dict[str, Any]) -> None:
        self.result = result
        self.query_kwargs: dict[str, Any] = {}

    def query(self, **kwargs: Any) -> dict[str, Any]:
        self.query_kwargs = kwargs
        return self.result


def _settings(**overrides: Any) -> Settings:
    values = {
        "rag_enabled": True,
        "rag_collection_name": "travel_knowledge",
        "rag_top_k": 5,
        "rag_min_score": 0.0,
    }
    values.update(overrides)
    return Settings(**values)


def test_returns_empty_list_when_rag_disabled() -> None:
    embedding_client = FakeEmbeddingClient(fail=True)
    service = KnowledgeSearchService(
        settings=_settings(rag_enabled=False),
        embedding_client=embedding_client,
        collection=FakeCollection({}),
    )

    assert service.search("Rome", ["food"], "local meals", 5) == []
    assert embedding_client.texts == []


def test_builds_search_text_from_destination_interests_and_query() -> None:
    embedding_client = FakeEmbeddingClient()
    collection = FakeCollection({"ids": [[]], "documents": [[]], "metadatas": [[]]})
    service = KnowledgeSearchService(
        settings=_settings(),
        embedding_client=embedding_client,
        collection=collection,
    )

    service.search("Rome", ["food", "hidden_gems"], "local meals", 5)

    assert len(embedding_client.texts) == 1
    search_text = embedding_client.texts[0]
    assert "Rome" in search_text
    assert "food" in search_text
    assert "hidden_gems" in search_text
    assert "local meals" in search_text


def test_returns_empty_list_when_collection_is_missing(monkeypatch) -> None:
    service = KnowledgeSearchService(
        settings=_settings(),
        embedding_client=FakeEmbeddingClient(),
    )
    monkeypatch.setattr(service, "_get_collection", lambda: None)

    assert service.search("Rome", ["food"], None, 5) == []


def test_filters_by_destination_metadata_and_maps_results() -> None:
    collection = FakeCollection(
        {
            "ids": [["rome:food.md:0", "paris:food.md:0", "rome:transport.md:0"]],
            "documents": [["Try carbonara.", "Try croissants.", "Validate bus tickets."]],
            "metadatas": [
                [
                    {"destination": "rome", "source": "food.md", "chunkIndex": 0},
                    {"destination": "paris", "source": "food.md", "chunkIndex": 0},
                    {"destination": "rome", "source": "transport.md", "chunkIndex": 0},
                ]
            ],
            "distances": [[0.0, 0.2, 3.0]],
        }
    )
    service = KnowledgeSearchService(
        settings=_settings(),
        embedding_client=FakeEmbeddingClient(),
        collection=collection,
    )

    results = service.search("Rome", ["food"], None, 5)

    assert collection.query_kwargs["where"] == {"destination": "rome"}
    assert len(results) == 2
    assert results[0].id == "rome:food.md:0"
    assert results[0].destination == "rome"
    assert results[0].source == "food.md"
    assert results[0].content == "Try carbonara."
    assert results[0].score == 1.0
    assert results[1].id == "rome:transport.md:0"
    assert results[1].score == 0.25


def test_embedding_failure_returns_empty_list_non_fatally() -> None:
    service = KnowledgeSearchService(
        settings=_settings(),
        embedding_client=FakeEmbeddingClient(fail=True),
        collection=FakeCollection({}),
    )

    assert service.search("Rome", ["food"], None, 5) == []
