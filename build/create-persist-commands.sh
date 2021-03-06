#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename "$0" .sh)"
progdir="$(dirname "$0")"
readonly progname progdir
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

: "${PERSIST_DIR:?need a persist directory}"
readonly PERSIST_DIR

persist_heroku="$PERSIST_DIR/heroku-deploy.sh"
persist_dockerhub="$PERSIST_DIR/docker-hub-deploy.sh"
persist_gcloud_login="$PERSIST_DIR/gcloud-login.sh"
persist_gcloud_registry="$PERSIST_DIR/gcr-deploy.sh"
readonly persist_heroku persist_heroku persist_gcloud_registry

mkdir -pv -- "$PERSIST_DIR"

finalize() {
  chmod 0755 "${1:?need a filename}"
  info "created: $1"
  ls -ld "$1"
}

# Retag is used to add extra tags, not replace the normal one.
# Consider if we want to change this.
retag() {
  [ -n "${RETAG:-}" ] || return 0
  local retagged
  retagged="${1:?}:${RETAG:?}"
  printf 'docker tag "%s" "%s"\ndocker push "%s"\n' \
    "$FULL_DOCKER_TAG" "$retagged" \
    "$retagged"
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
retag "$DOCKER_PROJECT" >>"$persist_dockerhub"

finalize "$persist_dockerhub"

printf >"$persist_gcloud_login" \
  '#!/bin/sh -eu\ngcloud config set project "%s"\ngcloud auth configure-docker </dev/null\n' \
  "$GCR_PROJECT"

finalize "$persist_gcloud_login"

printf >"$persist_gcloud_registry" \
  '#!/bin/sh -eu\ndocker tag "%s" "%s"\ndocker push "%s"\n' \
  "$FULL_DOCKER_TAG" "$GCR_REGISTRY_DOCKER_TAG" \
  "$GCR_REGISTRY_DOCKER_TAG"
retag "$GCR_REGISTRY_DOCKER_NAME" >>"$persist_gcloud_registry"

finalize "$persist_gcloud_registry"
