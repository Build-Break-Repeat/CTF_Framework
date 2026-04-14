#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bash "$SCRIPT_DIR/check_deps.sh" "$@"
bash "$SCRIPT_DIR/terraform_bootstrap.sh" "$@"
bash "$SCRIPT_DIR/terraform_deploy.sh" "$@"
bash "$SCRIPT_DIR/configure_firewall.sh"

echo "[*] Deployment Complete"
