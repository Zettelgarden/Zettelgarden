#!/bin/bash

set -e

source .env

# build.sh builds :latest from the current tree and deploys it to the PUBLIC
# instance on 192.168.0.93 (zettelgarden.com, /mnt/nas-2-fast-data/config/
# zettelgarden). Both that instance and zg-internal have been on SQLite since
# 2026-08-04, so master's Postgres-free :latest is safe to ship here. Run from
# master (or a release branch) — whatever tree you build is what goes live on
# zettelgarden.com.

# Build and push images locally
sudo docker compose build
sudo docker compose push

# SSH to remote server and deploy
ssh 192.168.0.93 "cd /mnt/nas-2-fast-data/config/zettelgarden && docker compose pull && docker compose up -d"
