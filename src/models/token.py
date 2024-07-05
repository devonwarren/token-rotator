from typing import Annotated, Literal, Optional, ClassVar
from pydantic import (
    BaseModel,
    Field,
    PastDatetime,
    AwareDatetime,
    SecretStr,
    StrictBool,
)
from models.crd import CRDMeta, CRDPrinterColumn


class TokenStatus(BaseModel):
    ready: Optional[bool] = None
    last_rotated: Optional[PastDatetime] = None
    expiration: Optional[AwareDatetime] = None


class Token(CRDMeta):
    "A generic token type"

    name: str = Field(description="The name of the token in k8s")
    # value: SecretStr = Field(description="The secret value of the token")
    # type: Literal['Secret'] = 'Secret'
    rotation_schedule: Annotated[
        str,
        Field(
            alias="rotationSchedule",
            description="A CronTab text representing when to run",
        ),
    ] = ""
    force_now: Annotated[
        bool,
        Field(
            default=False,
            alias="forceNow",
            description="Set to true to force a token refresh now",
        ),
    ] = False
    # status: Annotated[
    #     Optional[TokenStatus], Field(description="The current status info of the token")
    # ] = None
