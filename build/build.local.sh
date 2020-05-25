#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename "$0" .sh)"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

prebuild_sanity_check

go mod download

show_versions

ldflags="$(go_ldflags_stampversion)"
set -x
exec "$GO_CMD" build \
  -o "$BIN_NAME" \
  -tags "${BUILD_TAGS:-}" \
  -ldflags "$ldflags" \
  -v
