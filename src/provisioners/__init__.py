import abc


class BaseProvisioner(abc.ABC):
    def create_token(self, token_request: any) -> None:
        raise NotImplementedError

    def delete_token(self) -> None:
        raise NotImplementedError

    def check_token_validity(self) -> bool:
        raise NotImplementedError
