#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"

# Terraform execution
echo "[*] Running Terraform bootstrap"

cd "$TERRAFORM_DIR/bootstrap"
terraform init -input=false -upgrade

# Initial bootstrap of Terraform
sudo terraform apply -auto-approve
