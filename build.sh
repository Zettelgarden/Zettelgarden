#!/bin/bash

set -e

source .env

sudo docker compose build

#sudo docker compose push

cd /mnt/nas-2-fast-data/config/zettelgarden
sudo docker compose up -d
