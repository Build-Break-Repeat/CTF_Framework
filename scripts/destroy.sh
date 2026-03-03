#!/bin/bash
set -e
cd $(dirname "$0")/../terraform
sudo terraform destroy -auto-approve
