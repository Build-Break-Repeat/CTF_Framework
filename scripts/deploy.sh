#!/bin/bash
set -e

# Global Flags
NON_INTERACTIVE=false
AUTO_INSTALL=false

while [[ $# -gt 0 ]]; do
	case "$1" in
		--non-interactive|-n)
			NON_INTERACTIVE=true
			shift
			;;
		--auto-install|-a)
			AUTO_INSTALL=true
			shift
			;;
		--help|-h)
			echo "USAGE: ./deploy [--non-interactive] [--auto-install]"
			echo "--non-interactive | -n    No interactive prompts (no installs)"
			echo "--auto-install | -a       Automatically install any required packages"
			echo "--help | -h		Display this help message"
	esac
done

# Check if command exists: command_exists <command>
command_exists() {
	command -v "$1" > /dev/null 2>&1
}

# Install package: install_pkg <command>
install_pkg() {
	# Newer RHEL/CentOS systems
	if command_exists dnf; then
		sudo dnf install -y "$1"
	# Older RHEL/CentOS systems
	elif command_exists yum; then
		sudo yum install -y "$1"
	# Debian/Ubuntu systems
	elif command_exists apt; then
		sudo apt update -y && sudo apt install -y "$1"
	# Arch systems
	elif command_exists pacman; then
		sudo pacman -Sy --noconfirm "$1"
	# Alpine systems
	elif command_exists apk; then
		sudo apk add "$1"
	else
		echo "No supported Package Manager found to install $1"
		exit 1
	fi
}

# Docker install
install_docker() {
	if $NON_INTERACTIVE; then
                echo "[ERROR] Non-interactive mode, required package not installed: Docker"
        fi

	if ! $AUTO_INSTALL; then 
		read -p "[?] Docker is not installed. Would you like to install it now? (y/n): " RESPONSE
	else
		RESPONSE=y
	fi

        if [[ "$RESPONSE" == "y" || "$RESPONSE" == "Y" ]]; then
	        # Download the vendor's script for proper setup
	        curl -fsSL https://get.docker.com -o get-docker.sh

        	# Run it
	        sudo sh get-docker.sh
        else
                echo "[ERROR] Cannot proceed without Docker"
                exit 1
        fi
}

# Prompt user to install package: prompt_install <package>
prompt_install() {
	if $NON_INTERACTIVE; then
		echo "[ERROR] Non-interactive mode, required package not installed: $1"
	fi

	if ! $AUTO_INSTALL; then
		read -p "[?] $1 is not installed. Would you like to install it now? (y/n): " RESPONSE
	else
		RESPONSE=y
	fi

	if [[ "$RESPONSE" == "y" || "$RESPONSE" == "Y" ]]; then
		install_pkg $1
	else
		echo "[ERROR] Cannot proceed without $1"
		exit 1
	fi
}

# Dependency Checks
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
		exit 1;
	fi
	prompt_install "terraform"	
fi

# Docker service check
if ! systemctl is-active --quiet docker; then
	echo "[*] Docker service not running"
	
	if $NON_INTERACTIVE; then
		echo "[ERROR] Prompts disabled"
		exit 1
	elif $AUTO_INSTALL; then
		RESPONSE=y
	else
		read -p "[?] Docker service not running, start now? (y/n)" RESPONSE
	fi
	
	if [[ "$RESPONSE" == "y" || "$RESPONSE" == "Y" ]]; then
		sudo systemctl start docker
	else
		echo "[ERROR] Docker must be running."
		exit 1
	fi
fi

# Terraform execution
echo "[*] Running Terraform deployment"

cd $(dirname "$0")/../terraform

terraform init -input=false

if $NON_INTERACTIVE; then
	terraform apply -auto-approve
else
	terraform apply
fi

echo "[*] Deployment Complete"
