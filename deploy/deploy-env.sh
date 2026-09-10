#!/usr/bin/env bash
# Copy production env file to the deployment server.
# NEVER commit the .env file this script uploads.
set -euo pipefail

SERVER="ubuntu@13.52.254.25"
SSH_KEY="${1:-$HOME/.ssh/id_rsa}"
ENV_FILE="${2:-deploy/.env.production}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: $ENV_FILE not found. Create it from deploy/.env.production.example and fill in real secrets."
  exit 1
fi

echo "==> Uploading $ENV_FILE to /opt/readygeneration/.env..."
scp -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new "$ENV_FILE" "$SERVER:/opt/readygeneration/.env"

echo "==> Restarting API to pick up new env..."
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new "$SERVER" "sudo systemctl restart rg-api && sudo systemctl status rg-api --no-pager"
