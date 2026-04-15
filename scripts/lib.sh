#!/bin/bash
# Shared utilities for CTF scripts

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${SCRIPT_DIR}/../terraform"
PORT_STATE_FILE="${TERRAFORM_DIR}/firewall_ports.state"

# Global Flags
NON_INTERACTIVE=false
AUTO_INSTALL=false

# Check if command exists: command_exists <command>
command_exists() {
	command -v "$1" > /dev/null 2>&1
}

# Prompt user for confirmation: confirm "<message>"
# Returns 0 (yes) or 1 (no). Defaults to yes on Enter.
confirm() {
	read -r -p "$1 (Y/n): " RESPONSE
	RESPONSE=${RESPONSE:-y}
	[[ "$RESPONSE" == "y" || "$RESPONSE" == "Y" ]]
}

# Install package: install_pkg <package>
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
	else
		echo "No supported Package Manager found to install $1"
		exit 1
	fi
}

# Ensure current user is in the docker group and that the current session has docker group access.
ensure_docker_group() {
	local script_args=("$@")

	# Current session already has docker group access — nothing to do
	if id -Gn | grep -qw "docker"; then
		return 0
	fi

	# User is in the docker group but the current session hasn't loaded it yet
	if id -Gn "$USER" | grep -qw "docker"; then
		echo "[*] User '$USER' is in the docker group but the current session hasn't loaded it"
		echo "[*] Reloading group membership..."
		exec sg docker -c "bash $(printf '%q ' "$0" "${script_args[@]}")"
	fi

	# User is not in the docker group at all
	echo "[*] User '$USER' is not in the docker group"

	if $NON_INTERACTIVE; then
		echo "[ERROR] Non-interactive mode; '$USER' must be added to the docker group manually: sudo usermod -aG docker $USER"
		exit 1
	elif $AUTO_INSTALL || confirm "[?] Add '$USER' to the docker group?"; then
		sudo usermod -aG docker "$USER"
		echo "[*] Added '$USER' to the docker group. Reloading group membership..."
		exec sg docker -c "bash $(printf '%q ' "$0" "${script_args[@]}")"
	else
		echo "[ERROR] User must be in the docker group to run Docker commands without sudo."
		exit 1
	fi
}

# Docker install
install_docker() {
	if $NON_INTERACTIVE; then
		echo "[ERROR] Non-interactive mode, required package not installed: Docker"
		exit 1
	fi

	if $AUTO_INSTALL || confirm "[?] Docker is not installed. Would you like to install it now?"; then
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
		exit 1
	fi

	if $AUTO_INSTALL || confirm "[?] $1 is not installed. Would you like to install it now?"; then
		install_pkg "$1"
	else
		echo "[ERROR] Cannot proceed without $1"
		exit 1
	fi
}

# Parse global flags: parse_flags "$@"
parse_flags() {
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
				echo "USAGE: $0 [--non-interactive] [--auto-install]"
				echo "--non-interactive | -n    No interactive prompts (no installs)"
				echo "--auto-install | -a       Automatically install any required packages"
				echo "--help | -h               Display this help message"
				exit 0
				;;
			*) shift ;;
		esac
	done
}

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
record_port_state() {
	echo "$@" >> "$PORT_STATE_FILE"
}

# Remove Firewalld rules from state file
remove_firewalld_rules() {
	while read -r port; do
		echo "[*] Removing firewalld rule for port $port"
		sudo firewall-cmd --permanent --remove-port "${port}/tcp"
	done < "$PORT_STATE_FILE"

	sudo firewall-cmd --reload
}

# Remove UFW rules from state file
remove_ufw_rules() {
	while read -r port; do
		echo "[*] Removing UFW rule for port $port"
		sudo ufw delete allow "${port}/tcp"
	done < "$PORT_STATE_FILE"
}

# Remove IPTables rules from state file
remove_iptables_rules() {
	while read -r port; do
		echo "[*] Removing iptables rule for port $port"
		sudo iptables -D INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null || true
	done < "$PORT_STATE_FILE"
}

# Remove all firewall rules recorded during deploy
remove_firewall_rules() {
	if [ ! -f "$PORT_STATE_FILE" ]; then
		echo "[!] No port state file found"
		return
	fi

	FIREWALL=$(detect_firewall)

	case "$FIREWALL" in
		firewalld)
			remove_firewalld_rules
			;;
		ufw)
			remove_ufw_rules
			;;
		iptables)
			remove_iptables_rules
			;;
	esac

	rm -f "$PORT_STATE_FILE"
}

# Configure Firewall rules
config_firewall() {
	if ! systemctl is-active --quiet docker; then
		echo "[ERROR] Docker is not running. Cannot read container ports."
		exit 1
	fi

	echo "[*] Detect firewall type"
	FIREWALL=$(detect_firewall)

	if [[ "$FIREWALL" == "none" ]]; then
		echo "[!] No firewall detected, nothing to configure"
		return
	fi

	# For reference:
	# docker ps --format "{{.Ports}}"    Get list of ports
	# grep -oE '[0-9]+->'                Only get host ports
	# sed 's/->//'                       Remove the -> characters
	# grep -E '^[0-9]+$'                 Make sure there's no empty strings
	mapfile -t PORTS < <(docker ps --format "{{.Ports}}" | grep -oE '[0-9]+->' | sed 's/->//' | grep -E '^[0-9]+$')

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
