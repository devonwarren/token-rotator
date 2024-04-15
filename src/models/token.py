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


# the definition of the token to rotate
class Token(
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
            name="Spec",
            type="string",
            description="The cron spec defining the interval a CronJob is run",
            jsonPath=".spec.cronSpec",
        )
    ],
):
    name: str
    value: str
    type: str
    rotation_schedule: str = Field(alias="rotationSchedule")  # Crontab text definition
    force_now: bool = Field(default=False, alias="forceNow")
    # RotationStrategy: RotationStrategies = Field(default=RotationStrategies.IMMEDIATE)
    # Export: Export
