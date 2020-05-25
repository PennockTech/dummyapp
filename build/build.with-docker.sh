#!/usr/bin/env bash
# use bash so that we have arrays for constructing extra args safely

# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

progname="$(basename "$0" .sh)"
progdir="$(dirname "$0")"
# shellcheck source=build/common.lib.sh disable=SC2034
. "${progdir}/common.lib.sh" "$@"

prebuild_sanity_check

# Download all content into the vendor area outside Docker, so that Docker is
# not doing extra network fetches, and we can have a local cache for efficiency.
go mod vendor -v

if [ -d .git ]; then
  # If missing ssh command in image, Circle CI falls back to its own system for
  # git cloning, which can leave .pack files with permissions 0600, which
  # breaks across the user-id boundary when invoking docker (even if same uid
  # presented within each container, they'll be "different" after
  # normalization).
  # Fix 1: include ssh in all builder images.
  # Fix 2: stomp on .git permissions; this will re-open permissions if someone
  # is invoking this on their own system, but if the code is private, then the
  # parent dir will be private too, so nothing can get in because Unix requires
  # full transitive access.  This is why we _only_ chmod the .git area though.
  chmod -R go+rX .git
fi

if [ -n "$BUILD_TAGS" ]; then
  mkdir -pv -- "$(dirname "$DOCKER_MUTABLE_GO_TAGS")"
  printf > "$DOCKER_MUTABLE_GO_TAGS" '%s\n' "$BUILD_TAGS"
else
  rm -f -- "$DOCKER_MUTABLE_GO_TAGS"
fi

if [ -n "${EXTRACT_GO_VERSION_FROM_LABEL:-}" ]; then
  if [ -n "${DOCKER_BUILDER_IMAGE:-}" ]; then
    # This variable is used; it's picked up by the iteration for ARG FOO
    # below, but shellcheck misses that.
    # shellcheck disable=SC2034
    DOCKER_BUILDER_GOLANG_VERSION="$(docker_builder_golang_version "$DOCKER_BUILDER_IMAGE" "$EXTRACT_GO_VERSION_FROM_LABEL")"
  else
    warn "need DOCKER_BUILDER_IMAGE to use EXTRACT_GO_VERSION_FROM_LABEL='$EXTRACT_GO_VERSION_FROM_LABEL'"
  fi
elif [ -n "${DOCKER_BUILDER_IMAGE:-}" ]; then
  warn "we have DOCKER_BUILDER_IMAGE='$DOCKER_BUILDER_IMAGE' but no EXTRACT_GO_VERSION_FROM_LABEL"
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
