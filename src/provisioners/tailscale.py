from provisioners import BaseProvisioner
from pydantic import BaseModel, Field
from enum import Enum
import requests
from datetime import datetime
from typing import Optional


class TailscaleTokenCapabilities(str, Enum):
    CREATE_REUSABLE_DEVICE = "Create Reusable Device"
    CREATE_EPHEMERAL_DEVICE = "Create Ephemeral Device"
    CREATE_PREAUTHORIZED_DEVICE = "Create Preauthorized Device"


class TailscaleTokenRequest(BaseModel):
    """An instance of a Tailscale Token"""

    website: str

    description: str

    capabilities: list[TailscaleTokenCapabilities]

    expiration: Optional[datetime]


class TailscaleToken(BaseModel):
    """An instance of a Tailscale Token"""

    website: str

    description: str

    expiration: Optional[datetime]


class TailscaleTokenProvisioner(BaseProvisioner):
    base_url: str = "https://api.tailscale.com"
    client_id: str
    client_secret: str

    def create_token(self, token_request: TailscaleToken):
        # TODO: validate token is a GAT with pydantic?

        url = f"{self.base_url}/api/v2/CREATE_TOKEN"

        response = requests.post(url)
