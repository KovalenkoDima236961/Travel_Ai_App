import os
from functools import lru_cache

from pydantic import BaseModel, Field, field_validator


class Settings(BaseModel):
    app_env: str = "development"
    http_host: str = "0.0.0.0"
    http_port: int = Field(default=8000, ge=1, le=65535)
    log_level: str = "INFO"
    itinerary_generator_mode: str = "mock"
    ollama_base_url: str = "http://ollama:11434"
    ollama_model: str = "llama3.1:8b"
    ollama_timeout_seconds: float = Field(default=60, gt=0)
    ollama_temperature: float = Field(default=0.2, ge=0)
    ollama_num_predict: int = 2048
    ollama_fallback_to_mock: bool = True
    ollama_repair_enabled: bool = True
    ollama_repair_attempts: int = Field(default=1, ge=0)
    log_llm_payloads: bool = False
    destination_context_enabled: bool = True
    destination_context_dir: str = "app/data/destinations"

    @field_validator("ollama_repair_attempts")
    @classmethod
    def clamp_ollama_repair_attempts(cls, value: int) -> int:
        return min(value, 1)

    @property
    def allow_llm_payload_logging(self) -> bool:
        return self.log_llm_payloads and self.app_env.strip().lower() == "development"


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


def _env_float(name: str, default: float) -> float:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default
    return float(raw_value)


def _env_bool(name: str, default: bool) -> bool:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default

    normalized = raw_value.strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False

    raise ValueError(f"{name} must be a boolean value")


@lru_cache
def get_settings() -> Settings:
    return Settings(
        app_env=_env_string("APP_ENV", "development"),
        http_host=_env_string("HTTP_HOST", "0.0.0.0"),
        http_port=_env_int("HTTP_PORT", 8000),
        log_level=_env_string("LOG_LEVEL", "INFO").upper(),
        itinerary_generator_mode=_env_string("ITINERARY_GENERATOR_MODE", "mock"),
        ollama_base_url=_env_string("OLLAMA_BASE_URL", "http://ollama:11434"),
        ollama_model=_env_string("OLLAMA_MODEL", "llama3.1:8b"),
        ollama_timeout_seconds=_env_float("OLLAMA_TIMEOUT_SECONDS", 60),
        ollama_temperature=_env_float("OLLAMA_TEMPERATURE", 0.2),
        ollama_num_predict=_env_int("OLLAMA_NUM_PREDICT", 2048),
        ollama_fallback_to_mock=_env_bool("OLLAMA_FALLBACK_TO_MOCK", True),
        ollama_repair_enabled=_env_bool("OLLAMA_REPAIR_ENABLED", True),
        ollama_repair_attempts=_env_int("OLLAMA_REPAIR_ATTEMPTS", 1),
        log_llm_payloads=_env_bool("LOG_LLM_PAYLOADS", False),
        destination_context_enabled=_env_bool("DESTINATION_CONTEXT_ENABLED", True),
        destination_context_dir=_env_string("DESTINATION_CONTEXT_DIR", "app/data/destinations"),
    )
