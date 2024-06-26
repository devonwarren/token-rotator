from typing import ClassVar, Literal, Optional

from jsonpointer import JsonPointer
from pydantic import BaseModel, Field


class CustomResourceDefinitionNames(BaseModel):
    kind: str
    plural: str
    singular: Optional[str]
    categories: Optional[list[str]]
    listKind: Optional[str]
    shortNames: Optional[list[str]]


class CustomResourceDefinitionAdditionalPrinterColumn(BaseModel):
    jsonPath: str
    name: str
    type: Literal["integer", "number", "string", "boolean"]
    description: Optional[str] = ""
    format: Optional[str] = ""
    priority: Optional[int] = 0


class CustomResource(BaseModel):
    scope: ClassVar[str]
    group: ClassVar[str]
    names: ClassVar[CustomResourceDefinitionNames]
    additionalPrinterColumns: ClassVar[
        Optional[list[CustomResourceDefinitionAdditionalPrinterColumn]]
    ]

    def __init_subclass__(
        cls,
        *,
        scope: str,
        group: str,
        names: CustomResourceDefinitionNames | dict,
        additionalPrinterColumns: Optional[
            list[dict | CustomResourceDefinitionAdditionalPrinterColumn]
        ] = None,
    ):
        cls.scope = scope
        cls.group = group
        cls.names = CustomResourceDefinitionNames.model_validate(names)
        if additionalPrinterColumns:
            cls.additionalPrinterColumns = [
                CustomResourceDefinitionAdditionalPrinterColumn.model_validate(column)
                for column in additionalPrinterColumns
            ]
        else:
            cls.additionalPrinterColumns = []

    apiVersion: str = Field(
        ...,
        description="""APIVersion defines the versioned schema of this representation
of an object. Servers should convert recognized schemas to the latest
internal value, and may reject unrecognized values.
More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
""".replace("\n", " "),
    )
    kind: str = Field(
        ...,
        description="""Kind is a string value representing the REST resource this
object represents. Servers may infer this from the endpoint the client
submits requests to. Cannot be updated. In CamelCase.
More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
""".replace("\n", " "),
    )

    @classmethod
    def definition(cls) -> dict:
        "return the entire CRD definiton as a dict"
        schema = _resolve_refs(cls.model_json_schema())
        return {
            "apiVersion": "apiextensions.k8s.io/v1",
            "kind": "CustomResourceDefinition",
            "metadata": {"name": f"{cls.names.plural}.{cls.group}"},
            "spec": {
                "scope": cls.scope,
                "group": cls.group,
                "names": cls.names.model_dump(),
                "versions": [
                    {
                        "name": "v1",
                        "served": True,
                        "storage": True,
                        "schema": {"openAPIV3Schema": schema},
                        "additionalPrinterColumns": [
                            col.model_dump()
                            for col in cls.additionalPrinterColumns or []
                        ],
                    }
                ],
            },
        }

    class Config:
        json_schema_extra = {"x-kubernetes-preserve-unknown-fields": True}


_sentinel = object()


def _resolve_refs(schema: dict, part=_sentinel):
    """Resolve references in schema generated by pydantic.

    Does not support remote or cyclical references.
    """
    if part is _sentinel:
        part = schema
    if not isinstance(part, (dict, list)):
        return part
    if isinstance(part, list):
        return [_resolve_refs(schema, item) for item in part]
    if "$ref" in part:
        return _resolve_refs(
            schema, JsonPointer(part["$ref"].lstrip("#")).resolve(schema)
        )
    return {
        key: _resolve_refs(schema, value)
        for key, value in part.items()
        if key != "definitions"
    }
