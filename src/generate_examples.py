from models.token import Token
import yaml

sample_token = Token(
    name="sample-token",
    value="test",
    type="test",
    rotationSchedule="",
)

with open("examples/sample-token.yaml", "w+") as yaml_file:
    yaml.dump_all([sample_token.model_dump_json()], yaml_file)