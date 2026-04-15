#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"
ensure_docker_group "$@"

bash "$SCRIPT_DIR/terraform_destroy_challenges.sh"
bash "$SCRIPT_DIR/terraform_destroy_bootstrap.sh"
bash "$SCRIPT_DIR/remove_firewall.sh"

echo "[*] Destroy Complete"
