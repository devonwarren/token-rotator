from models.token import Token
import yaml

token_crd = Token.definition()

# export the token crd definition
with open("deploy/crds/token.yaml", "w+") as yaml_file:
    yaml.dump_all([token_crd], yaml_file)
