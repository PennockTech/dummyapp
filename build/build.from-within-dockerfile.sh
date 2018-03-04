#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename -s .sh "$0")"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

show_versions
echo

go_tags="docker${BUILD_TAGS:+ }${BUILD_TAGS:-}"
ld_flags="$(go_ldflags_stampversion) -s"
output_file="$(binary_handoff_path)"

set -x
cd "$CTXPROJDIR"
export CGO_ENABLED=0 GOOS="${DOCKER_GOOS}"
exec "$GO_CMD" build \
  -tags "$go_tags" \
  -ldflags "$ld_flags" \
  -a -installsuffix docker-nocgo \
  -o "$output_file" \
  "$GO_PROJECT_PATH"
