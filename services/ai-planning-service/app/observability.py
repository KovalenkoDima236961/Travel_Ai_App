from __future__ import annotations

import logging
import re
import time
import uuid
from collections.abc import Awaitable, Callable

from fastapi import Request, Response
from prometheus_client import (
    CONTENT_TYPE_LATEST,
    Counter,
    Gauge,
    Histogram,
    generate_latest,
)
from starlette.responses import Response as StarletteResponse

HEADER_REQUEST_ID = "X-Request-ID"
HEADER_CORRELATION_ID = "X-Correlation-ID"
logger = logging.getLogger(__name__)

HTTP_REQUESTS = Counter(
    "http_requests_total",
    "Total HTTP requests handled by service.",
    ["service", "method", "route", "status"],
)
HTTP_DURATION = Histogram(
    "http_request_duration_seconds",
    "HTTP request duration by service, method, route, and status.",
    ["service", "method", "route", "status"],
)
HTTP_IN_FLIGHT = Gauge(
    "http_requests_in_flight",
    "In-flight HTTP requests by service, method, and route.",
    ["service", "method", "route"],
)
AI_REQUESTS = Counter(
    "ai_requests_total",
    "Total AI planning requests.",
    ["operation", "result", "mode"],
)
AI_DURATION = Histogram(
    "ai_request_duration_seconds",
    "AI planning request duration.",
    ["operation", "result", "mode"],
    buckets=(1, 5, 10, 30, 60, 120, 300, 600, 1200),
)
AI_VALIDATION_FAILURES = Counter(
    "ai_validation_failures_total",
    "Total AI request validation failures.",
    ["operation"],
)
AI_REPAIR_ATTEMPTS = Counter(
    "ai_repair_attempts_total",
    "Total AI repair attempts.",
    ["operation", "result"],
)

_UUID_SEGMENT = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)


async def request_context_middleware(
    request: Request,
    call_next: Callable[[Request], Awaitable[StarletteResponse]],
) -> StarletteResponse:
    request_id = request.headers.get(HEADER_REQUEST_ID) or str(uuid.uuid4())
    correlation_id = request.headers.get(HEADER_CORRELATION_ID) or request_id
    request.state.request_id = request_id
    request.state.correlation_id = correlation_id

    route = _route_label(request)
    in_flight_route = route
    HTTP_IN_FLIGHT.labels("ai-planning-service", request.method, in_flight_route).inc()
    started_at = time.monotonic()
    status = 500
    try:
        response = await call_next(request)
        status = response.status_code
        return response
    finally:
        duration = time.monotonic() - started_at
        route = _route_label(request)
        HTTP_IN_FLIGHT.labels("ai-planning-service", request.method, in_flight_route).dec()
        HTTP_REQUESTS.labels("ai-planning-service", request.method, route, str(status)).inc()
        HTTP_DURATION.labels("ai-planning-service", request.method, route, str(status)).observe(
            duration
        )
        logger.info(
            "http_request",
            extra={
                "service": "ai-planning-service",
                "method": request.method,
                "path": request.url.path,
                "route": route,
                "status": status,
                "durationMs": int(duration * 1000),
                "requestId": request_id,
                "correlationId": correlation_id,
            },
        )
        if "response" in locals():
            response.headers[HEADER_REQUEST_ID] = request_id
            response.headers[HEADER_CORRELATION_ID] = correlation_id


def metrics_response() -> Response:
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


def record_ai_request(operation: str, result: str, mode: str, duration_seconds: float) -> None:
    AI_REQUESTS.labels(operation, result, mode).inc()
    AI_DURATION.labels(operation, result, mode).observe(duration_seconds)


def record_ai_validation_failure(operation: str) -> None:
    AI_VALIDATION_FAILURES.labels(operation).inc()


def record_ai_repair_attempt(operation: str, result: str) -> None:
    AI_REPAIR_ATTEMPTS.labels(operation, result).inc()


def _route_label(request: Request) -> str:
    route = request.scope.get("route")
    path = getattr(route, "path", None)
    if isinstance(path, str) and path:
        return path
    return _sanitize_path(request.url.path)


def _sanitize_path(path: str) -> str:
    parts = []
    for part in path.split("/"):
        if not part:
            continue
        if _UUID_SEGMENT.match(part):
            parts.append("{uuid}")
        elif part.isdigit():
            parts.append("{number}")
        else:
            parts.append(part)
    return "/" + "/".join(parts) if parts else "/"
