#!/usr/bin/env bash
# use bash so that we have arrays for constructing extra args safely

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

if [ -n "$BUILD_TAGS" ]; then
  mkdir -pv -- "$(dirname "$DOCKER_MUTABLE_GO_TAGS")"
  printf > "$DOCKER_MUTABLE_GO_TAGS" '%s\n' "$BUILD_TAGS"
else
  rm -f -- "$DOCKER_MUTABLE_GO_TAGS"
fi

declare -a extra_args

# Used to stop after a certain FROM stage (target) in the Dockerfile.
if [ -n "${MAKE_DOCKER_TARGET:-}" ]; then
  extra_args+=( --target "${MAKE_DOCKER_TARGET:?}" )
fi

# Any `ARG FOO` in the Dockerfile can be overridden through
# an environment variable DOCKER_FOO passed into us,.
for arg in $(docker_available_ARGs); do
  envvar="DOCKER_$arg"
  if [ -n "${!envvar:-}" ]; then
    extra_args+=( --build-arg "${arg}=${!envvar}" )
  fi
done

# We had support for $GO_PARENTDIR being set from outside, but I think
# with the shell setup, we might be moving away from needing that in future.
# For now, the Dockerfile still expects this override, but we can make
# it optional.
if [ -n "${GO_PARENTDIR:-}" ]; then
  extra_args+=( --build-arg "GO_PARENTDIR=$GO_PARENTDIR" )
fi

# The EXTRA_DOCKER_BUILD_ARGS is deliberately unquoted.
# shellcheck disable=SC2086
trace_cmd "$DOCKER_CMD" build \
  --tag "$FULL_DOCKER_TAG" \
  --file "$DOCKERFILE" \
  --build-arg "APP_VERSION=$REPO_VERSION" \
  --build-arg "GO_BUILD_TAGS=$BUILD_TAGS" \
  "${extra_args[@]}" \
  ${EXTRA_DOCKER_BUILD_ARGS:-} \
  .

rm -f -- "$DOCKER_MUTABLE_GO_TAGS"

# For Circle CI with workflows, where we try deploys from different stages
# within the workflow, we need to be able to make our Docker image available to
# docker within those stages, and do so by persisting it to a file.
if [ -n "${DIND_PERSIST_FILE:-}" ]; then
  mkdir -pv -- "$(dirname "$DIND_PERSIST_FILE")"
  trace_cmd "$DOCKER_CMD" save -o "$DIND_PERSIST_FILE" "$FULL_DOCKER_TAG"
  ls -ld -- "$DIND_PERSIST_FILE"
fi
