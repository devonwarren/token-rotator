from pydantic import BaseModel, Field
from enum import Enum


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
class Token(BaseModel):
    Name: str
    RotationSchedule: str  # Crontab text definition
    ForceNow: bool = Field(default=False)
    RotationStrategy: RotationStrategies = Field(default=RotationStrategies.IMMEDIATE)
    Export: Export
