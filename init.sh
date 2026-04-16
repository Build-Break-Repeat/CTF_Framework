#!/bin/bash
set -euo pipefail

echo "Configuring git hooks..."
git config core.hooksPath .githooks
chmod +x .githooks/post-merge
echo "Git hooks configured."

echo "Downloading latest ctfctl binary..."
curl -fsSL "https://github.com/Build-Break-Repeat/CTF_Framework/releases/latest/download/ctfctl" -o ./ctfctl
chmod +x ./ctfctl
echo "ctfctl installed."
