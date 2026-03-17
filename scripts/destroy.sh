#!/bin/bash
set -e

# Set directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${SCRIPT_DIR}/../terraform"
PORT_STATE_FILE="${TERRAFORM_DIR}/firewall_ports.state"

cd "$TERRAFORM_DIR"

# Destroy the environment
sudo terraform destroy -auto-approve

# Get firewall type
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
	
	# Not found
	echo "none"
}

# Config Firewalld
config_firewalld() {
	while read -r port; do
		echo "[*] Removing firewalld rule for port $port"
		sudo firewall-cmd --permanent --remove-port ${port}/tcp
	done < "$PORT_STATE_FILE"
	
	sudo firewall-cmd --reload
}

# Config UWF
config_ufw() {
	while read -r port; do
                echo "[*] Removing UFW rule for port $port"
                sudo ufw delete allow ${port}/tcp
        done < "$PORT_STATE_FILE"
}

# Config IPTables
config_iptables() {
	while read -r port; do
                echo "[*] Removing iptables rule for port $port"
                sudo iptables -D INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null || true
        done < "$PORT_STATE_FILE"
}

#Remove the firewall rules
remove_firewall_rules() {
	if [ ! -f "$PORT_STATE_FILE" ]; then
		echo "[!] No port state file found"
		return
	fi

	FIREWALL=$(detect_firewall)

	case "$FIREWALL" in
		firewalld)
			config_firewalld
			;;
		ufw)
			config_ufw
			;;
		iptables)
			config_iptables
			;;
	esac

	rm -f "$PORT_STATE_FILE"
}

# Run it
remove_firewall_rules
