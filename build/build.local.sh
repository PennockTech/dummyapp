#!/bin/sh -eu
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename -s .sh "$0")"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

prebuild_sanity_check

if should_dep_fetch; then
  ensure_have_dep
  trace_cmd "$DEP_CMD" ensure -v
fi

show_versions

ldflags="$(go_ldflags_stampversion)"
set -x
exec "$GO_CMD" build \
  -o "$BIN_NAME" \
  -tags "${BUILD_TAGS:-}" \
  -ldflags "$ldflags" \
  -v
