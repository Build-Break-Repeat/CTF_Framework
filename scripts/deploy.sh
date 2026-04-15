#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"

bash "$SCRIPT_DIR/check_deps.sh" "$@"
ensure_docker_group "$@"
bash "$SCRIPT_DIR/terraform_bootstrap.sh" "$@"
python3 "$SCRIPT_DIR/ctfd_bootstrap.py"
bash "$SCRIPT_DIR/terraform_deploy.sh" "$@"
bash "$SCRIPT_DIR/configure_firewall.sh"

echo "[*] Deployment Complete"
