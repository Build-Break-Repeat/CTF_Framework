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

Current workflow:
1. Challenge names come from terraform/challenges.json.
2. Generated files are written under flags/<challenge-name>/team#.txt.
3. injectflags.py copies those team files into matching Docker containers.
4. By default, flags go to /flags inside each container.

To run ctfctl:
sudo apt install golang-go -y
git clone -b build-control-script https://github.com/Build-Break-Repeat/CTF_Framework.git
cd CTF_Framework
chmod +x ctfctl

./ctfctl deploy
./ctfctl destroy
./ctfctl rebuild
./ctfctl reset
./ctfctl bootstrap
./ctfctl challenge list
