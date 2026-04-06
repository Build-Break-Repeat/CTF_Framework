# Build Break Repeat - Infrastructure Foundation

## Requirements
- Docker
- Terraform
- Git
- Curl
- Wget
- Python 3


## Deploy
./scripts/deploy.sh

## Destroy
./scripts/destroy.sh

## Flags
Generate team-based flags:
python3 scripts/createflags.py 3

Generate team-based flags with a different preset:
python3 scripts/createflags.py 3 --preset lab

List available presets:
python3 scripts/createflags.py --list-presets

Inject generated flags into the running challenge containers:
python3 scripts/injectflags.py
