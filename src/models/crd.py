from typing import Literal, Optional, ClassVar
from pydantic import BaseModel, Field


class CRDPrinterColumn(BaseModel):
    "A column to be printed when doing a kubectl get"

    # TODO: better validation strings
    json_path: str
    name: str
    type: Literal["integer", "number", "string", "boolean", "date"]
    description: Optional[str] = ""
    format: Optional[str] = ""
    priority: Optional[int] = 0


class CRDModel(BaseModel):
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


class CRDMeta(BaseModel):
    "A common base model for CRDs that adds the ability to export the schema"

    # info for CRD definition when exporting schema
    crd_meta: ClassVar[CRDModel] = CRDModel(
        scope="Namespaced",
        group="token-rotator.org",
        kind="Token",
        singular="token",
        plural="tokens",
        list_kind="TokenList",
        printed_columns=[
            CRDPrinterColumn(
                json_path=".status.ready",
                name="Status",
                type="string",
                description="If the token is in a ready state",
            ),
        ],
    )

    def get_crd_meta():
        "Returns the CRD meta information"
        return crd_meta.model_dump()
    
    def get_crd_yaml() -> dict:
        "Returns the CRD YAML Kubernetes manifest"
        
        crd = crd_meta
        return {
            "apiVersion": "apiextensions.k8s.io/v1",
            "kind": "CustomResourceDefinition",
            "metadata": {
                "name": f"{crd.plural.lower()}.{crd.group}",
                "labels": {
                    "app": "token-rotator",
                    "role": "token-resource",
                },
            },
            "spec": {
                "group": crd.group,
                "scope": crd.scope,
                "names": {
                    "kind": crd.kind,
                    "singular": crd.singular,
                    "plural": crd.plural,
                    "listKind": crd.list_kind,
                },
                "versions": [
                    {
                        "name": "v1",
                        "served": True,
                        "storage": True,
                        "schema": {
                            "openAPIV3Schema": {
                                "type": "object",
                                "properties": self.model_json_schema()["properties"],
                            },
                        },
                    }
                ],
            },
        }
    
    
    