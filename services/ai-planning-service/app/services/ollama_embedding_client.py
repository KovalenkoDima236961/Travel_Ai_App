from typing import Any

import httpx

from app.config import Settings


class OllamaEmbeddingError(RuntimeError):
    """Raised when the Ollama embeddings API cannot provide a usable embedding."""


class OllamaEmbeddingClient:
    def __init__(self, settings: Settings, http_client: httpx.Client | None = None) -> None:
        self._settings = settings
        self._http_client = http_client

    def embed(self, text: str) -> list[float]:
        payload = {
            "model": self._settings.ollama_embedding_model,
            "prompt": text,
        }

        try:
            response = self._post_to_ollama(payload)
        except httpx.HTTPError as exc:
            raise OllamaEmbeddingError("Ollama embedding request failed") from exc

        if response.status_code < 200 or response.status_code >= 300:
            raise OllamaEmbeddingError(
                f"Ollama embeddings API returned HTTP {response.status_code}"
            )

        try:
            body = response.json()
        except ValueError as exc:
            raise OllamaEmbeddingError("Ollama embeddings API returned invalid JSON") from exc

        return self._parse_embedding(body)

    def _post_to_ollama(self, payload: dict[str, Any]) -> httpx.Response:
        endpoint = f"{self._settings.ollama_base_url.rstrip('/')}/api/embeddings"
        timeout = self._settings.ollama_embedding_timeout_seconds

        if self._http_client is not None:
            return self._http_client.post(endpoint, json=payload, timeout=timeout)

        with httpx.Client(timeout=timeout) as client:
            return client.post(endpoint, json=payload)

    def _parse_embedding(self, body: dict[str, Any]) -> list[float]:
        raw_embedding = body.get("embedding")
        if not isinstance(raw_embedding, list) or not raw_embedding:
            raise OllamaEmbeddingError(
                "Ollama embeddings API response is missing a non-empty 'embedding' field"
            )

        embedding: list[float] = []
        for value in raw_embedding:
            if not isinstance(value, int | float):
                raise OllamaEmbeddingError("Ollama embeddings API returned a non-numeric value")
            embedding.append(float(value))

        return embedding
