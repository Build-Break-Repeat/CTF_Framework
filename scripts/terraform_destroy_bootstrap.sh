#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"
ensure_docker_group "$@"

echo "[*] Running Terraform destroy (bootstrap)"

cd "$TERRAFORM_DIR/bootstrap"

# Destroy the environment
sudo terraform destroy -auto-approve

# Remove API key and admin password
rm -f "$TERRAFORM_DIR/ctfd_token.txt"
rm -f "$SCRIPT_DIR/../admin_password.txt"
