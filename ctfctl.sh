#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CTFCTL_DIR="$SCRIPT_DIR/cmd/ctfctl"
cd "$CTFCTL_DIR" || exit 1

exec go run . "$@"

# this is just a wrapper script that allows ctfctl to function without compiling
