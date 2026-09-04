#!/usr/bin/env bash
# Deploy readygeneration-backend to AWS Ubuntu server
# No Docker Desktop required — cross-compiles binary locally.
# Usage: ./deploy/deploy.sh [ssh-key-path]
#   e.g. ./deploy/deploy.sh ~/.ssh/readygeneration.pem
set -euo pipefail

SERVER="ubuntu@13.52.254.25"
APP_DIR="/opt/readygeneration"
SSH_KEY="${1:-$HOME/.ssh/id_rsa}"
SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=accept-new"
SCP="scp -i $SSH_KEY -r"

echo "==> Cross-compiling for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-w -s" -o /tmp/rg-api ./cmd/api

echo "==> Stopping API service before upload..."
$SSH $SERVER "sudo systemctl stop rg-api || true"

echo "==> Uploading binary + migrations..."
$SCP /tmp/rg-api       $SERVER:$APP_DIR/api
$SCP migrations/       $SERVER:$APP_DIR/migrations
$SCP deploy/docker-compose.prod.yml $SERVER:$APP_DIR/docker-compose.yml
$SCP deploy/nginx.conf $SERVER:$APP_DIR/nginx.conf
$SCP deploy/rg-api.service $SERVER:/tmp/rg-api.service

echo "==> Installing systemd service..."
$SSH $SERVER "sudo mv /tmp/rg-api.service /etc/systemd/system/rg-api.service \
  && sudo systemctl daemon-reload \
  && sudo systemctl enable rg-api"

echo "==> Activating nginx site..."
$SSH $SERVER "sudo ln -sf $APP_DIR/nginx.conf /etc/nginx/sites-enabled/readygeneration \
  && sudo rm -f /etc/nginx/sites-enabled/default \
  && sudo nginx -t \
  && sudo systemctl reload nginx"

echo "==> Setting binary as executable..."
$SSH $SERVER "chmod +x $APP_DIR/api"

echo "==> Starting PostgreSQL via Docker..."
$SSH $SERVER "cd $APP_DIR && sudo docker compose up -d postgres"

echo "==> Waiting for Postgres to be ready..."
$SSH $SERVER "sleep 6"

echo "==> Running database migrations..."
$SSH $SERVER "set -a && . $APP_DIR/.env && set +a && MIGRATE_ONLY=true $APP_DIR/api"

echo "==> (Re)starting API service..."
$SSH $SERVER "sudo systemctl restart rg-api && sudo systemctl status rg-api --no-pager"

echo ""
echo "==========================================="
echo " Deployed! https://api.readygeneration.com"
echo "==========================================="
