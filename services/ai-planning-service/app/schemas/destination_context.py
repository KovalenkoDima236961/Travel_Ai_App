from typing import Annotated

from pydantic import BaseModel, Field, StringConstraints

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
