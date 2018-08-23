#!/bin/sh
# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

#
# Invoke me with VERBOSE=2 in env to see the commands actually being run.
#

set -eu
progname="$(basename "$0" .sh)"
progdir="$(dirname "$0")"
# We expect 'local' in sh, even though non-POSIX.
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

: "${REFORMAT:=false}"

for F in "$BUILD_DIR"/*.sh "$BUILD_DIR/version"; do
  info "$F"
  short="$(basename "$F")"
  should_shfmt=true

  # Codes and reason to disable:
  #  SC2039: non-POSIX: We expect 'local' in sh, even though non-POSIX
  # For the library _only_:
  #  SC2034: the common library routinely sets variables which are unused
  #          within the library; should be disabled when sourcing
  #  SC1008: we're using "#!/bin/echo"
  #  SC2096: we're using "#!/bin/echo" with some parameters, and the shellcheck
  #          explanation is wrong anyway.
  DISABLE="SC2039"

  case "$short" in
  common.lib.sh)
    DISABLE="${DISABLE},SC2034,SC1008,SC2096"
    ;;
  build.with-docker.sh)
    verbose_n 0 "skipping shfmt for '${short}': it breaks on array+=( ... )"
    should_shfmt=false
    ;;
  esac

  if $should_shfmt; then
    if "$REFORMAT"; then
      trace_cmd shfmt -i 2 -w "$F"
    else
      output="$(trace_cmd shfmt -i 2 -l "$F")"
      if [ -n "$output" ]; then
        warn "shfmt wishes to reformat"
      fi
    fi
  fi

  trace_cmd shellcheck -e "$DISABLE" -a -x "$F" || true
done
