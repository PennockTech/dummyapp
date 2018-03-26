#!/usr/bin/env bash
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename -s .sh "$0")"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

# Here, insert any environment variables needed to run the app
# : "${DATABASE_URL:?}"

: "${LOCATION:=local-docker on $(hostname -s)}"

declare -a envs args dflags

dflags=(--rm --read-only --tty --publish-all)

args=("/${BIN_NAME:?}")
args+=(-log.json)
envs+=(LOCATION)

for ename in "${envs[@]}"; do
  [ -n "${!ename:-}" ] || continue
  dflags+=(-e "${ename}=${!ename}")
done

did=$("$DOCKER_CMD" run --detach "${dflags[@]}" "$FULL_DOCKER_TAG" "${args[@]}")

echo "Docker ID: $did"
"$DOCKER_CMD" port "$did"
port="$("$DOCKER_CMD" port "$did" | head -n 1)"
port="${port##*:}"

"$DOCKER_CMD" ps -f "id=$did"

if [ -n "$DOCKER_MACHINE_NAME" ]; then
  docker-machine ip "$DOCKER_MACHINE_NAME"
  ip="$(docker-machine ip "$DOCKER_MACHINE_NAME")"
  case $ip in
  *:*) ip="[$ip]" ;;
  esac
  echo
  echo "http://${ip}:${port}/"
else
  echo
  echo "http://localhost:$port/"
fi

echo
echo "Resuming connection to: ${did}"

# I want these expanded now, shellcheck, you silly billy
# shellcheck disable=SC2064
trap "'$DOCKER_CMD' kill '$did' || true" EXIT INT TERM

"$DOCKER_CMD" attach "$did"
