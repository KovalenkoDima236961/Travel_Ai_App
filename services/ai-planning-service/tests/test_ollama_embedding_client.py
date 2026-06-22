import json

import httpx
import pytest

from app.config import Settings
from app.services.ollama_embedding_client import OllamaEmbeddingClient, OllamaEmbeddingError


def _settings() -> Settings:
    return Settings(
        ollama_base_url="http://ollama:11434",
        ollama_embedding_model="nomic-embed-text",
        ollama_embedding_timeout_seconds=3,
    )


def test_successful_embedding_response_returns_float_list() -> None:
    captured: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["body"] = json.loads(request.content)
        return httpx.Response(200, json={"embedding": [1, 0.5, -0.25]})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        client = OllamaEmbeddingClient(settings=_settings(), http_client=http_client)
        embedding = client.embed("Rome food tips")

    assert captured["path"] == "/api/embeddings"
    assert captured["body"] == {"model": "nomic-embed-text", "prompt": "Rome food tips"}
    assert embedding == [1.0, 0.5, -0.25]


def test_non_2xx_embedding_response_raises_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, json={"error": "failed"})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        client = OllamaEmbeddingClient(settings=_settings(), http_client=http_client)

        with pytest.raises(OllamaEmbeddingError, match="HTTP 500"):
            client.embed("Rome")


def test_missing_embedding_response_raises_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"model": "nomic-embed-text"})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        client = OllamaEmbeddingClient(settings=_settings(), http_client=http_client)

        with pytest.raises(OllamaEmbeddingError, match="embedding"):
            client.embed("Rome")


def test_embedding_connection_error_is_wrapped() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("connection refused", request=request)

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        client = OllamaEmbeddingClient(settings=_settings(), http_client=http_client)

        with pytest.raises(OllamaEmbeddingError, match="request failed"):
            client.embed("Rome")
