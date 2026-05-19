#!/usr/bin/env bash
set -Eeuo pipefail

APP_DIR="${APP_DIR:-/home/tvr/sportzal}"
COMPOSE_DIR="${COMPOSE_DIR:-/home/tvr/fencing}"
BRANCH="${BRANCH:-main}"

request_file=/tmp/sportzal-deploy.requested
lock_file=/tmp/sportzal-deploy.lock

touch "$request_file"

exec 9>"$lock_file"
if ! flock -n 9; then
  echo "Deploy is already running; queued the latest version"
  exit 0
fi

while true; do
  rm -f "$request_file"

  echo "Syncing sportzal repositories"
  git -C "$APP_DIR/backend" fetch --prune origin "$BRANCH"
  git -C "$APP_DIR/backend" reset --hard "origin/$BRANCH"
  git -C "$APP_DIR/frontend" fetch --prune origin "$BRANCH"
  git -C "$APP_DIR/frontend" reset --hard "origin/$BRANCH"

  cd "$COMPOSE_DIR"

  echo "Building sportzal images before replacing containers"
  docker compose build sport-backend sport-frontend

  echo "Replacing sportzal containers after successful build"
  if docker compose up --help | grep -q -- '--wait'; then
    docker compose up -d --no-build --remove-orphans --wait sport-backend sport-frontend
  else
    docker compose up -d --no-build --remove-orphans sport-backend sport-frontend
  fi

  docker compose ps sport-backend sport-frontend

  if [ ! -f "$request_file" ]; then
    sleep 2
  fi

  if [ ! -f "$request_file" ]; then
    break
  fi

  echo "Another deploy request arrived; deploying the latest version"
done
