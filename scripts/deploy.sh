#!/bin/bash
set -e

# Global Flags
NON_INTERACTIVE=false
AUTO_INSTALL=false

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${SCRIPT_DIR}/../terraform"
PORT_STATE_FILE="${TERRAFORM_DIR}/firewall_ports.state"

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
		read -r -p "[?] Docker is not installed. Would you like to install it now? (y/n): " RESPONSE
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
		read -r -p "[?] $1 is not installed. Would you like to install it now? (y/n): " RESPONSE
	else
		RESPONSE=y
	fi

	if [[ "$RESPONSE" == "y" || "$RESPONSE" == "Y" ]]; then
		install_pkg "$1"
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
		read -r -p "[?] Docker service not running, start now? (y/n)" RESPONSE
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

cd "$TERRAFORM_DIR"

terraform init -input=false -upgrade

if $NON_INTERACTIVE; then
	sudo terraform apply -auto-approve
else
	sudo terraform apply
fi

# Detect firewall type
detect_firewall() {
	# Firewalld (CentOS/RHEL)
	if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld; then
		echo "firewalld"
		return
	fi

	# UFW (Ubuntu/Debian)
	if command -v ufw >/dev/null 2>&1 && ufw status | grep -q "Status: active"; then
		echo "ufw"
		return
	fi

	# IPTables (fallback)
	if command -v iptables >/dev/null 2>&1; then
		echo "iptables"
		return
	fi

	# None detected
	echo "none"
}

# Configure Firewalld
config_firewalld() {
	for port in "$@"; do
		echo "[*] Opening port $port via firewalld"
		sudo firewall-cmd --permanent --add-port "${port}/tcp" && record_port_state "${port}"
	done

	echo "[*] Reloading firewalld with rule changes"
	sudo firewall-cmd --reload
}

# Configure UFW
config_ufw() {
	for port in "$@"; do
		echo "[*] Opening port $port via UFW"
		sudo ufw allow "${port}/tcp" && record_port_state "${port}"
	done
}

# Configure IPTables
config_iptables() {
	for port in "$@"; do
		echo "[*] Opening port $port via iptables"
		# Check for existing port, create if not exists
		if ! sudo iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null; then
		    sudo iptables -I INPUT -p tcp --dport "$port" -j ACCEPT
		    record_port_state "$port"
		fi
	done
}

# Record port state for destruction phase
record_port_state(){
	echo "$@" >> "$PORT_STATE_FILE"
}

# Configure Firewall rules
config_firewall() {
	echo "[*] Detect firewall type"
	FIREWALL=$(detect_firewall)

	if [[ "$FIREWALL" == "none" ]]; then
		echo "[!] No firewall detected, nothing to configure"
		return
	fi

	# For reference:
	# docker ps --format "{{.Ports}}"    Get list of ports
	# grep -oE '[0-9]+->'  		     Only get host ports
	# sed 's/->//'			     Remove the -> characters
	# grep -E '^[0-9]+$'		     Make sure there's no empty strings
	mapfile -t PORTS < <(docker ps --format "{{.Ports}}" | grep -oE '[0-9]+->' | sed 's/->//' | grep -E '^[0-9]+$') 

	echo "[*] Detected container ports: $PORTS"

	case "$FIREWALL" in
		firewalld)
			config_firewalld "${PORTS[@]}"
			;;
		ufw)	
			config_ufw "${PORTS[@]}"
			;;
		iptables)
			config_iptables "${PORTS[@]}"
			;;

	esac
}

config_firewall

echo "[*] Deployment Complete"
