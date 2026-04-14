#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

echo "[*] Running Terraform destroy (challenges)"

cd "$TERRAFORM_DIR/challenges"

# Destroy challenges
sudo terraform destroy -auto-approve
