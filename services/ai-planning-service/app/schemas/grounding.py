from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator


class GroundingModel(BaseModel):
    model_config = ConfigDict(populate_by_name=True)


class GroundingDestination(GroundingModel):
    id: str | None = None
    canonical_name: str = Field(alias="canonicalName", min_length=1, max_length=160)
    country_code: str | None = Field(default=None, alias="countryCode", max_length=2)
    country_name: str | None = Field(default=None, alias="countryName", max_length=160)
    aliases: list[str] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)


class GroundingPlace(GroundingModel):
    id: str | None = None
    canonical_name: str = Field(alias="canonicalName", min_length=1, max_length=240)
    category: str = Field(min_length=1, max_length=80)
    tags: list[str] = Field(default_factory=list)
    typical_duration_minutes: int | None = Field(
        default=None, alias="typicalDurationMinutes", ge=5, le=720
    )
    price_level: str | None = Field(default=None, alias="priceLevel")
    outdoor: bool | None = None
    rain_friendly: bool | None = Field(default=None, alias="rainFriendly")
    best_time_of_day: list[str] = Field(default_factory=list, alias="bestTimeOfDay")
    confidence: float = Field(ge=0, le=1)
    source_key: str | None = Field(default=None, alias="sourceKey")
    source_url: str | None = Field(default=None, alias="sourceUrl")


class GroundingDocument(GroundingModel):
    id: str | None = None
    title: str = Field(min_length=1, max_length=300)
    summary: str = Field(min_length=1, max_length=2000)
    source_key: str | None = Field(default=None, alias="sourceKey")
    confidence: float = Field(default=0.7, ge=0, le=1)


class GroundingContext(GroundingModel):
    status: Literal["available", "partial", "unavailable"] = "unavailable"
    destination: GroundingDestination | None = None
    places: list[GroundingPlace] = Field(default_factory=list, max_length=20)
    documents: list[GroundingDocument] = Field(default_factory=list, max_length=8)
    retrieval_warnings: list[str] = Field(
        default_factory=list, alias="retrievalWarnings", max_length=12
    )
    generated_at: datetime | None = Field(default=None, alias="generatedAt")
    knowledge_version: str | None = Field(default=None, alias="knowledgeVersion", max_length=128)

    @field_validator("places")
    @classmethod
    def unique_places(cls, places: list[GroundingPlace]) -> list[GroundingPlace]:
        seen: set[str] = set()
        result: list[GroundingPlace] = []
        for place in places:
            key = (place.id or place.canonical_name).casefold()
            if key in seen:
                continue
            seen.add(key)
            result.append(place)
        return result


GroundingSource = Literal["grounded", "provider", "generic", "model_suggested"]
