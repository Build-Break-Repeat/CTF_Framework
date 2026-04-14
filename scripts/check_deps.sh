#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"

# GIT
if ! command_exists git; then
	prompt_install "git"
fi

# Curl
if ! command_exists curl; then
	prompt_install "curl"
fi

# Wget
if ! command_exists wget; then
	prompt_install "wget"
fi

# Docker
if ! command_exists docker; then
	install_docker
fi

# Terraform
if ! command_exists terraform; then
	if command_exists apt; then
		wget -O - https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
		echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(grep -oP '(?<=UBUNTU_CODENAME=).*' /etc/os-release || lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
	elif command_exists dnf; then
		prompt_install "yum-utils"
		sudo yum-config-manager --add-repo https://rpm.releases.hashicorp.com/RHEL/hashicorp.repo
	else
		echo "[ERROR] Unable to install Terraform. Manual install may be necessary"
		exit 1
	fi
	prompt_install "terraform"
fi

# Docker service check
if ! systemctl is-active --quiet docker; then
	echo "[*] Docker service not running"

	if $NON_INTERACTIVE; then
		echo "[ERROR] Prompts disabled"
		exit 1
	elif $AUTO_INSTALL || confirm "[?] Docker service not running, start now?"; then
		sudo systemctl start docker
	else
		echo "[ERROR] Docker must be running."
		exit 1
	fi
fi

# Docker group check
ensure_docker_group "$@"
