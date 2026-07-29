from __future__ import annotations

from app.model_router.metadata import routing_metadata_from_request


def safe_request_routing_metadata(request: object) -> dict[str, object]:
    return routing_metadata_from_request(request)

