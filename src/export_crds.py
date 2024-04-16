from models.token import Token, TokenStatus
from models.crd import CRDMeta, CRDPrinterColumn
import yaml
from icecream import ic

# Serialize the crd_meta instance to JSON
crd_meta = Token.crd_meta
print(crd_meta)

# Define the CRD YAML structure
crd_yaml = {
    "apiVersion": "apiextensions.k8s.io/v1",
    "kind": "CustomResourceDefinition",
    "metadata": {
        "name": f"{crd_meta.plural.lower()}.{crd_meta.group}",
        "labels": {
            "app": "token-rotator",
            "role": "token-resource",
        },
    },
    "spec": {
        "group": crd_meta.group,
        "scope": crd_meta.scope,
        "names": {
            "kind": crd_meta.kind,
            "singular": crd_meta.singular,
            "plural": crd_meta.plural,
            "listKind": crd_meta.list_kind,
        },
        "versions": [
            {
                "name": "v1",
                "served": True,
                "storage": True,
                "schema": {
                    "openAPIV3Schema": {
                        "type": "object",
                        "properties": Token.model_json_schema()["properties"],
                    },
                },
            }
        ],
    },
}

ic(crd_yaml)

# # Convert the dictionary to YAML format
crd_yaml_str = yaml.dump(crd_yaml, default_flow_style=False)

# print(crd_yaml_str)


# export the token crd definition
with open("deploy/crds/token.yaml", "w+") as yaml_file:
    yaml.dump_all([crd_yaml], yaml_file, default_flow_style=False)
