#!/usr/bin/env bash
# One-shot VPS deploy (Ubuntu/Debian): ./deploy/vps.sh
set -euo pipefail
if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sh
fi
[ -f .env ] || { echo "Create .env first (cp sample.env .env && nano .env)"; exit 1; }
docker compose up -d --build
docker compose logs -f --tail=50
