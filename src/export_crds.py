from models.token import Token, TokenStatus
from models.crd import CRDMeta, CRDPrinterColumn
import yaml
from icecream import ic

# Serialize the crd_meta instance to JSON
crd_meta = Token.crd_meta
print(crd_meta)


ic(crd_yaml)

# # Convert the dictionary to YAML format
crd_yaml_str = yaml.dump(crd_yaml, default_flow_style=False)

# print(crd_yaml_str)


# export the token crd definition
with open("deploy/crds/token.yaml", "w+") as yaml_file:
    yaml.dump_all([crd_yaml], yaml_file, default_flow_style=False)
