#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename -s .sh "$0")"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

: "${PERSIST_DIR:?need a persist directory}"

persist_heroku="$PERSIST_DIR/heroku-deploy.sh"
persist_dockerhub="$PERSIST_DIR/docker-hub-deploy.sh"

mkdir -pv -- "$PERSIST_DIR"

finalize() {
  chmod 0755 "${1:?need a filename}"
  info "created: $1"
  ls -ld "$1"
}

# For these files, we assume that the 'docker' command first in $PATH is
# correct for the point where the scripts will be invoked and that the
# pre-persist command-names doesn't carry across.  If something else is needed,
# add support.

printf >"$persist_heroku" \
  '#!/bin/sh -eu\ndocker tag "%s" "%s"\ndocker push "%s"\n' \
  "$FULL_DOCKER_TAG" "$HEROKU_REGISTRY_DOCKER_TAG" \
  "$HEROKU_REGISTRY_DOCKER_TAG"

finalize "$persist_heroku"

printf >"$persist_dockerhub" \
  '#!/bin/sh -eu\ndocker push "%s"\n' \
  "$FULL_DOCKER_TAG"
if [ -n "${RETAG:-}" ]; then
  retagged="${DOCKER_PROJECT}:${RETAG}"
  printf >>"$persist_dockerhub" \
    'docker tag "%s" "%s"\ndocker push "%s"\n' \
    "$FULL_DOCKER_TAG" "$retagged" \
    "$retagged"
fi

finalize "$persist_dockerhub"
