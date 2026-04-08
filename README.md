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
1. Challenge names come from terraform/config.json.
2. Generated files are written under flags/<challenge-name>/team#.txt.
3. injectflags.py copies those team files into matching Docker containers.
4. By default, flags go to /flags inside each container.

Simple demo challenge:
1. Generate flags: python3 scripts/createflags.py 1
2. Inject flags: python3 scripts/injectflags.py
3. Open http://<server-ip>:8010 and read the hint on the page.
4. Check the page source, then visit /robots.txt.
5. Follow that hint to /flag.txt and submit the team flag.
