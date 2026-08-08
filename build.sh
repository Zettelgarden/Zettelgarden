#!/bin/bash

set -e

source .env

# build.sh builds :latest from the current tree and deploys it to the remote
# instance (a self-hosted server you control — the default below is the
# operator's own box; override DEPLOY_HOST / DEPLOY_PATH for yours).
# Run from master (or a release branch) — whatever tree you build is what
# goes live.

# Deploy target (override in .env or on the command line)
DEPLOY_HOST="${DEPLOY_HOST:-192.168.0.93}"
DEPLOY_PATH="${DEPLOY_PATH:-/mnt/nas-2-fast-data/config/zettelgarden}"

# Build and push images locally
sudo docker compose build
sudo docker compose push

# SSH to remote server and deploy
ssh "$DEPLOY_HOST" "cd '$DEPLOY_PATH' && docker compose pull && docker compose up -d"
