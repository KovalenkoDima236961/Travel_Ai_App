"""Non-sensitive build metadata exposed by the service version endpoint."""

import os
from dataclasses import asdict, dataclass


def _value(name: str, default: str) -> str:
    value = os.getenv(name, "").strip()
    return value or default


@dataclass(frozen=True)
class VersionInfo:
    service: str
    version: str
    gitSha: str
    buildTime: str
    environment: str
    apiContractVersion: str


def get_version_info() -> VersionInfo:
    version = _value("APP_VERSION", "dev")
    return VersionInfo(
        service="ai-planning-service",
        version=version,
        gitSha=_value("GIT_SHA", "unknown"),
        buildTime=_value("BUILD_TIME", "unknown"),
        environment=_value("APP_ENV", "local"),
        apiContractVersion=version,
    )


def version_payload() -> dict[str, str]:
    return asdict(get_version_info())
