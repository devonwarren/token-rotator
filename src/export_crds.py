from models.token import TokenCRD
import yaml

token_crd = TokenCRD.definition()

# export the token crd definition
with open("deploy/crds/token.yaml", "w+") as yaml_file:
    yaml.dump_all([token_crd], yaml_file)
