#!/bin/bash

set -e

source .env

# ⚠️ GUARD — read before overriding.
# build.sh builds :latest from master and deploys it to the PUBLIC instance
# on 192.168.0.93 (zettelgarden.com, /mnt/nas-2-fast-data/config/zettelgarden).
# That instance is still on Postgres (+ B2 storage); master has been
# Postgres-free since commit 54046de9 (2026-08-04). Deploying :latest now would
# push a Postgres-free binary at a Postgres-backed site and take the public
# site down.
#
# Do NOT override until the public instance is migrated to SQLite + local
# storage. See:
#   docs/plans/2026-08-04-sqlite-schema-sync-and-pg-retirement.md  ("Remaining work")
# Once that lands, override with:  ZG_PUBLIC_DEPLOY_CONFIRMED=1 ./build.sh
if [ "${ZG_PUBLIC_DEPLOY_CONFIRMED:-}" != "1" ]; then
  echo "⚠️  build.sh deploys :latest to the PUBLIC instance on 192.168.0.93," >&2
  echo "    which is still on Postgres+B2. Master has been Postgres-free since" >&2
  echo "    commit 54046de9 — deploying now would break zettelgarden.com." >&2
  echo "    Migrate the public instance first; see" >&2
  echo "    docs/plans/2026-08-04-sqlite-schema-sync-and-pg-retirement.md." >&2
  echo "    Override once cleared: ZG_PUBLIC_DEPLOY_CONFIRMED=1 ./build.sh" >&2
  exit 1
fi

# Build and push images locally
sudo docker compose build
sudo docker compose push

# SSH to remote server and deploy
ssh 192.168.0.93 "cd /mnt/nas-2-fast-data/config/zettelgarden && docker compose pull && docker compose up -d"
