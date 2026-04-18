#!/bin/bash
set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_flags "$@"
ensure_docker_group "$@"

echo "[*] Running Terraform challenges deployment"

cd "$TERRAFORM_DIR/challenges"
terraform init -input=false -upgrade

if $NON_INTERACTIVE || $AUTO_INSTALL; then
	sudo terraform apply -auto-approve
else
	sudo terraform apply
fi
