import os
from functools import lru_cache

from pydantic import BaseModel, Field


class Settings(BaseModel):
    app_env: str = "development"
    http_host: str = "0.0.0.0"
    http_port: int = Field(default=8000, ge=1, le=65535)
    log_level: str = "INFO"


def _env_string(name: str, default: str) -> str:
    value = os.getenv(name)
    if value is None or value.strip() == "":
        return default
    return value.strip()


def _env_int(name: str, default: int) -> int:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default
    return int(raw_value)


@lru_cache
def get_settings() -> Settings:
    return Settings(
        app_env=_env_string("APP_ENV", "development"),
        http_host=_env_string("HTTP_HOST", "0.0.0.0"),
        http_port=_env_int("HTTP_PORT", 8000),
        log_level=_env_string("LOG_LEVEL", "INFO").upper(),
    )
