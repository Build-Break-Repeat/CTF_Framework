#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"

echo "[*] Running Terraform challenges deployment"

cd "$TERRAFORM_DIR/challenges"
terraform init -input=false -upgrade

if $NON_INTERACTIVE; then
	sudo terraform apply -auto-approve
else
	sudo terraform apply
fi
