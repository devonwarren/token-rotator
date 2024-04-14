from lightkube import KubernetesApiClient, KubernetesObject
from models import Token


def create_token(token: Token):
    # Create a KubernetesApiClient
    client = KubernetesApiClient()

    # Define the Kubernetes object
    k8s_object = KubernetesObject(
        apiVersion="your.api.group/v1",
        kind="Token",
        metadata={"name": "example-token"},  # Set the name of the token
        spec=token.dict(),
    )

    # Create the resource
    client.create(k8s_object, namespace="default")  # Change namespace if needed


# Example usage
if __name__ == "__main__":
    token = Token(
        RotationSchedule="0 0 * * *",  # Run at midnight every day
        ForceNow=False,
        RotationStrategy="Immediate",
        Export=Export(
            Type="example", Name="example-name", Namespace="example-namespace"
        ),
    )
    create_token(token)
