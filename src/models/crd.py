from typing import Literal, Optional
from pydantic import BaseModel, Field


class CRDPrinterColumn(BaseModel):
    "A column to be printed when doing a kubectl get"

    json_path: str = Field(alias="jsonPath")
    name: str
    type: Literal["integer", "number", "string", "boolean", "date"]
    description: Optional[str] = ""
    format: Optional[str] = ""
    priority: Optional[int] = 0


class CRDMeta(BaseModel):
    "A list of CustomResourceDefinition attributes"

    scope: Optional[Literal["Namespaced", "Cluster"]] = Field(
        default="Namespaced",
        description="If the CRD is namespace scoped or cluster-wide",
    )
    group: Optional[str] = Field(
        default="token-rotator.org",
        description="The kubernetes group the resource is a part of",
    )
    kind: str = Field(
        description="The name of the CRD object", pattern=r"^[A-Za-z\-]*$"
    )
    singular: str = Field(description="The singular naming of the object")
    plural: str = Field(description="The plural naming of the objects")
    list_kind: str = Field(description="The list naming of the objects")
    short_names: Optional[list[str]] = Field(
        default=None, description="A list of short names to use with kubectl"
    )
    categories: Optional[list[str]] = Field(
        default=None, description="Any categories to be included in such as `all`"
    )
    printed_columns: Optional[list[CRDPrinterColumn]] = None
