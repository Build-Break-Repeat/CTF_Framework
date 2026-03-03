#!/bin/bash
set -e
cd $(dirname "$0")/../terraform
terraform destroy -auto-approve
