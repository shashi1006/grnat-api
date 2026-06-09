#!/usr/bin/env bash
# One-time server bootstrap for Ubuntu 24.04
# Run as: ssh ubuntu@13.52.254.25 'bash -s' < deploy/setup.sh
set -euo pipefail

echo "==> Installing Docker..."
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker ubuntu

echo "==> Installing nginx..."
sudo apt-get install -y nginx

echo "==> Creating app directory..."
sudo mkdir -p /opt/readygeneration
sudo chown ubuntu:ubuntu /opt/readygeneration

echo "==> Enabling nginx..."
sudo systemctl enable nginx

echo "Setup complete. Run deploy.sh to push the first build."
