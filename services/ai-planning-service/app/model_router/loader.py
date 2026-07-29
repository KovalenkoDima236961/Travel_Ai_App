from __future__ import annotations

from app.config import Settings
from app.services.adapter_runtime import validate_adapter_runtime


def validate_deployment_loadable(settings: Settings, deployment_key: str | None) -> bool:
    if not deployment_key:
        return True
    status = validate_adapter_runtime(settings)
    if not status.adapter_enabled:
        return True
    return status.is_ready

