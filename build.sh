#!/bin/bash

set -e

source .env

# Build and push images locally
sudo docker compose build
sudo docker compose push

# SSH to remote server and deploy
ssh 192.168.0.93 "cd /mnt/nas-2-fast-data/config/zettelgarden && docker compose pull && docker compose up -d"
