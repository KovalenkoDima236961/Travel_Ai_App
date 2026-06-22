from typing import Annotated

from pydantic import BaseModel, ConfigDict, Field, StringConstraints

NonEmptyString = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]


class DestinationContext(BaseModel):
    destination: NonEmptyString
    aliases: list[str] = Field(default_factory=list)
    localTips: list[str] = Field(default_factory=list)
    hiddenGems: list[str] = Field(default_factory=list)
    foodTips: list[str] = Field(default_factory=list)
    avoid: list[str] = Field(default_factory=list)
    transportTips: list[str] = Field(default_factory=list)
    budgetTips: list[str] = Field(default_factory=list)


class DestinationContextSummary(BaseModel):
    destination: NonEmptyString
    aliases: list[str] = Field(default_factory=list)
    source: str = "file"


class DestinationContextListResponse(BaseModel):
    items: list[DestinationContextSummary]


class DestinationContextPromptPreviewResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    destination_context_found: bool = Field(alias="destinationContextFound")
    destination_context: DestinationContext | None = Field(default=None, alias="destinationContext")
    prompt: str


class DestinationContextNotFoundResponse(BaseModel):
    error: str
