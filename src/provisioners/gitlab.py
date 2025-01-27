from provisioners import BaseProvisioner
from pydantic import BaseModel, Field
from enum import Enum
import requests
from datetime import datetime
from typing import Optional


class GitlabAccessTokenPermissions(str, Enum):
    """A list of permissions that could be applied to a Gitlab Access Token"""

    READ_API = "read_api"
    READ_REGISTRY = "read_registry"
    WRITE_REGISTRY = "write_registry"


class GitlabAccessToken(BaseModel):
    """An instance of a Gitlab Access Token"""

    project_id: int

    description: str

    permissions: list[GitlabAccessTokenPermissions]

    expiration: Optional[datetime]


class GitlabAccessTokenProvisioner(BaseProvisioner):
    # where api calls will be directed (could be different if using self hosted gitlab)
    base_url: str = "https://gitlab.com"
    provisioner_token: str

    def create_token(self, token_request: GitlabAccessToken):
        # TODO: validate token is a GAT with pydantic?

        url = f"{self.base_url}/api/v4/CREATE_TOKEN"

        response = requests.post(url)
