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
    neighborhood: str | None = None
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

    # Quality metadata supplied by Trip Service. grounding_strength is the
    # field the prompt acts on: "strong" records may be named confidently,
    # "weak" records must be marked needsPlaceReview. Records the knowledge
    # store rejected or merged never arrive here at all.
    quality_score: float = Field(default=0.0, alias="qualityScore", ge=0, le=1)
    freshness_score: float = Field(default=0.0, alias="freshnessScore", ge=0, le=1)
    review_status: str | None = Field(default=None, alias="reviewStatus", max_length=32)
    grounding_strength: Literal["strong", "weak", "excluded"] = Field(
        default="weak", alias="groundingStrength"
    )
    opening_hours_summary: str | None = Field(
        default=None, alias="openingHoursSummary", max_length=200
    )
    warnings: list[str] = Field(default_factory=list, max_length=6)

    @property
    def is_strong(self) -> bool:
        return self.grounding_strength == "strong"


class DestinationCoverage(GroundingModel):
    """Coverage tells generation when to stop asserting specific places.

    Low coverage produces generic activities and a partial-quality result
    rather than plausible-sounding invented place names.
    """

    place_count: int = Field(default=0, alias="placeCount", ge=0)
    high_quality_place_count: int = Field(default=0, alias="highQualityPlaceCount", ge=0)
    category_coverage: float = Field(default=0.0, alias="categoryCoverage", ge=0, le=1)
    freshness_coverage: float = Field(default=0.0, alias="freshnessCoverage", ge=0, le=1)
    coordinate_coverage: float = Field(default=0.0, alias="coordinateCoverage", ge=0, le=1)
    opening_hours_coverage: float = Field(
        default=0.0, alias="openingHoursCoverage", ge=0, le=1
    )
    coverage_score: float = Field(default=0.0, alias="coverageScore", ge=0, le=1)
    status: Literal["available", "partial", "limited", "unavailable"] = "unavailable"
    warnings: list[str] = Field(default_factory=list, max_length=8)


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
    coverage: DestinationCoverage | None = None
    attributions: list[str] = Field(default_factory=list, max_length=12)
    generated_at: datetime | None = Field(default=None, alias="generatedAt")
    knowledge_version: str | None = Field(default=None, alias="knowledgeVersion", max_length=128)

    @property
    def strong_places(self) -> list[GroundingPlace]:
        return [place for place in self.places if place.is_strong]

    @property
    def weak_places(self) -> list[GroundingPlace]:
        return [place for place in self.places if place.grounding_strength == "weak"]

    @field_validator("places")
    @classmethod
    def drop_excluded_places(cls, places: list[GroundingPlace]) -> list[GroundingPlace]:
        """Defence in depth.

        Trip Service already excludes rejected and merged records in SQL. This
        second check means a caller that ignores that contract still cannot put
        an excluded record in front of the model.
        """
        return [place for place in places if place.grounding_strength != "excluded"]

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
