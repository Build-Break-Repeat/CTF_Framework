# Build Break Repeat - Infrastructure Foundation

An IaC tool for deploying and managing a CTF competition environment on a single Linux host.

### What does `ctfctl` do?
`ctfctl` handles the full lifecycle of a CTF event. It spins up a CTFd scoreboard, deploys vulnerable challenge containers, generates and injects unique flags per team, and configures a firewall, all from a single command. When the event is over, it tears everything down for you just as easily.

## Requirements
- Linux (CentOS/Ubuntu recommended)
- Docker
- Terraform
- Git
- Curl
- Wget
- Python 3

Installing the basics:

```bash
sudo apt update
sudo apt install git -y
sudo apt install curl -y
```

## Setup

```bash
git clone https://github.com/Build-Break-Repeat/CTF_Framework.git
cd CTF_Framework
bash init.sh
```

## Usage/Deployment

```bash
# Enter Sudo for Administrator Privilege
sudo su

# Deploy CTFCTL and Follow on Screen Prompts
./ctfctl deploy
```

## Additional Usage Commands

```bash
# Check running containers and their URLs
./ctfctl status

# Tear everything down
./ctfctl destroy

# Destroy then redeploy
./ctfctl rebuild
```

## Configuration

Challenges and event settings are managed through `config.json`. Use the CLI to edit instead of editing the file directly:

```bash
# Add, edit, remove, or list challenges
./ctfctl challenge add
./ctfctl challenge list
./ctfctl challenge edit

# View event settings
./ctfctl event show

# Edit event name, team count, admin credentials
./ctfctl event edit
```

## Flags

Flags are generated per team and injected automatically during `deploy`. To run manually:

```bash
./ctfctl flags generate
./ctfctl flags inject
```
