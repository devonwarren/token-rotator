from typing import Literal, Optional
from pydantic import (
    BaseModel,
    Field,
    PastDatetime,
    AwareDatetime,
    SecretStr,
    PrivateAttr,
)
from models.crd import CRDMeta, CRDPrinterColumn


class TokenStatus(BaseModel):
    ready: Optional[bool] = None
    last_rotated: Optional[PastDatetime] = None
    expiration: Optional[AwareDatetime] = None


class Token(BaseModel):
    "A generic token type"

    name: str = Field(description="The name of the token in k8s")
    value: SecretStr = Field(description="The secret value of the token")
    # type: Literal['Secret'] = 'Secret'
    rotation_schedule: Optional[str] = Field(
        alias="rotationSchedule", description="A CronTab text representing when to run"
    )
    force_now: Optional[bool] = Field(
        default=False,
        alias="forceNow",
        description="Set to true to force a token refresh now",
    )
    status: Optional[TokenStatus | dict]

    # internal attributes
    _crd_meta = CRDMeta(
        scope="Namespaced",
        group="token-rotator.org",
        kind="Token",
        singular="token",
        plural="tokens",
        list_kind="TokenList",
    )
