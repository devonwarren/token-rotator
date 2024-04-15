from pydantic import BaseModel, Field
from enum import Enum
from models.crd import CustomResource, CustomResourceDefinitionAdditionalPrinterColumn


# the type of export type to save token into
class ExportTypes(str, Enum):
    SECRET = "Secret"


class RotationStrategies(str, Enum):
    IMMEDIATE = "Immediate"
    KEEP_OLD = "KeepOld"


# the export info for how the token gets saved
class Export(BaseModel):
    Type: ExportTypes
    Name: str
    Namespace: str
    Annotations: dict[str, str] = Field(default={})

class TokenSpec(BaseModel):
    name: str
    value: str
    type: str
    rotation_schedule: str = Field(alias="rotationSchedule")  # Crontab text definition
    force_now: bool = Field(default=False, alias="forceNow")


# the definition of the token to rotate
class TokenCRD(
    CustomResource,
    scope="Namespaced",
    group="token-rotator.org",
    names={
        "kind": "Token",
        "plural": "tokens",
        "singular": "token",
        "listKind": "TokenList",
        "shortNames": [],
        "categories": ["all", "token"],
    },
    additionalPrinterColumns=[
        CustomResourceDefinitionAdditionalPrinterColumn(
            name="Status",
            type="string",
            description="If the token is currently valid without any issues",
            jsonPath='.status.conditions[?(@.type=="Ready")].status',
        ),
        CustomResourceDefinitionAdditionalPrinterColumn(
            name="Expiration",
            type="date",
            description="A timestamp of when this token is set to expire",
            jsonPath=".status.expirationTimestamp",
        ),
        CustomResourceDefinitionAdditionalPrinterColumn(
            name="Last Rotated",
            type="date",
            description="A timestamp of when this token was last rotated",
            jsonPath=".status.lastRotated",
        ),
    ],
):
    spec: TokenSpec
    # RotationStrategy: RotationStrategies = Field(default=RotationStrategies.IMMEDIATE)
    # Export: Export
