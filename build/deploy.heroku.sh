#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename "$0" .sh)"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

printf '%s\n' "$BUILD_TAGS" | xargs -n1 | grep -qs '^heroku$' || die "missing 'heroku' in build tags"

info "heroku: pushing '$FULL_DOCKER_TAG' to '$HEROKU_REGISTRY_DOCKER_TAG'"
"$DOCKER_CMD" tag "$FULL_DOCKER_TAG" "$HEROKU_REGISTRY_DOCKER_TAG"
exec "$DOCKER_CMD" push "$HEROKU_REGISTRY_DOCKER_TAG"
