from typing import Annotated

from pydantic import BaseModel, Field, StringConstraints, field_validator

NonEmptyString = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]


class KnowledgeChunk(BaseModel):
    id: NonEmptyString
    destination: NonEmptyString
    source: NonEmptyString
    content: NonEmptyString
    metadata: dict[str, str | int | float | bool] = Field(default_factory=dict)


class KnowledgeSearchRequest(BaseModel):
    destination: NonEmptyString
    interests: list[str] = Field(default_factory=list)
    query: str | None = None
    topK: int = Field(default=5, ge=1, le=10)

    @field_validator("interests", mode="before")
    @classmethod
    def default_interests(cls, value: object) -> object:
        if value is None:
            return []
        return value

    @field_validator("interests")
    @classmethod
    def normalize_interests(cls, value: list[str]) -> list[str]:
        return [item.strip().lower() for item in value if item.strip()]

    @field_validator("query")
    @classmethod
    def normalize_query(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip()
        return normalized or None


class KnowledgeSearchResult(BaseModel):
    id: NonEmptyString
    destination: NonEmptyString
    source: NonEmptyString
    content: NonEmptyString
    score: float | None = None
    metadata: dict[str, str | int | float | bool] = Field(default_factory=dict)


class KnowledgeSearchResponse(BaseModel):
    items: list[KnowledgeSearchResult]
