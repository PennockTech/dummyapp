#!/bin/echo you should source me
# Note that shellcheck complains about that idiom; shellcheck is unaware.

# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

# Preamble: {{{
set -eu
#
# Local convention: any line starting `: "${VAR_NAME:=` should be parseable by
# a fairly simple regex to extract defaults by something which doesn't speak
# shell, so for lines which are not introducing new variables overrideable
# thru environ, I'm starting them `: : ${` (even if they're not `:=`).

: : "${progname:?}" "${progdir:?}"

# The caller should have passed their own argv onto us when sourcing.  We
# take env-KEY=VALUE parameters and handle those, ignoring everything else.
# We need to do this before setting values which can be overridden from env.
for param; do
  case "$param" in
  env-*)
    full="${param#env-}"
    key="${full%%=*}"
    value="${full#*=}"
    export "${key}"="$value"
    ;;
  esac
done
unset param full key value
# Preamble: }}}

# ============================8< EDIT THESE >8============================

: "${GITHUB_PROJECT:=PennockTech/dummyapp}"
: "${DOCKER_PROJECT:=pennocktech/dummyapp}"
: "${HEROKU_APP:=pt-dummy-app}"
: "${GO_PROJECT_PATH:=go.pennock.tech/dummyapp}"
: "${BIN_NAME:=dummyapp}"

# Should we be defaulting to heroku here, or leave that only to CircleCI?
: "${BUILD_TAGS:=heroku}"

# ============================8< EDIT THESE >8============================

# Whether or not to inherit the timestamp is interesting: which is more likely
# to have accurate and trusted time, the container, or the system triggering
# the build in the container?  For now, generate as close to the build as
# possible, ignoring env.
: "${BUILD_TIMESTAMP:=$(date -u "+%Y-%m-%d %H:%M:%SZ")}"

# Let the caller override name/path/whatever, eg to build with a different
# version of Go.
: "${GIT_CMD:=git}"
: "${GO_CMD:=go}"
: "${DEP_CMD:=dep}"
: "${DOCKER_CMD:=docker}"

# Diagnostic functions: {{{

# I really don't like how these single-liners get blown out by shfmt but
# having consistency is worth it.  I think.  Maybe.
: "${VERBOSE:=0}"
warn_count=0
if [ -n "${NOCOLOR:-}" ]; then
  _stderr_colored() {
    shift
    printf >&2 '%s: %s\n' "$progname" "$*"
  }
else
  # shellcheck disable=SC1117
  _stderr_colored() {
    local color="$1"
    shift
    printf >&2 "\033[${color}m%s: \033[1m%s\033[0m\n" "${progname}" "$*"
  }
fi
info() { _stderr_colored 32 "$@"; }
warn() {
  _stderr_colored 31 "$@"
  warn_count=$((warn_count + 1))
}
die() {
  _stderr_colored 31 "$@"
  exit 1
}
verbose_n() {
  [ "$VERBOSE" -ge "$1" ] || return 0
  shift
  _stderr_colored 36 "$@"
}
verbose() { verbose_n 1 "$@"; }
report_exit() {
  if [ "$warn_count" -gt 0 ]; then
    warn "saw ${warn_count} warnings"
  fi
  exit "$1"
}

trace_cmd() {
  verbose_n 2 invoking: "$*"
  "$@"
}

# Diagnostic functions: }}}

# Derived Variables: {{{

# Docker 'golang:N.NN' images have runtime user root, HOME=/home/root, cwd=/go
# and GOPATH=/go with permissions 0777.
# Most other things use $HOME/go with sane permissions.
# We default to the golang images but want to be easy to use with anything
# else.
firstGopath="${GOPATH%%:*}"
: "${HOME:=/home/$(id -un)}"
: : "${firstGopath:=${HOME}/go}"

CTXPROJDIR="${firstGopath}/src/${GO_PROJECT_PATH}"

REPO_DIR="$("$GIT_CMD" rev-parse --show-toplevel)"
BUILD_DIR="$("$GIT_CMD" -C "$progdir" rev-parse --show-prefix)"
BUILD_DIR="${BUILD_DIR%/}"
DOCKERFILE="${BUILD_DIR}/Dockerfile"

# This needs to be within the context passed to the Docker builder, so the
# filesystem can't really be read-only, but it's a bit weird to have to modify
# the source tree on a per-build basis without sub-dirs.  So we support
# moving this and making the parent dir read-only (in theory, not confirmed).
: "${DOCKER_MUTABLE_GO_TAGS:=build/.docker-go-tags}"

LOCAL_OS="$(uname)"
: "${DOCKER_GOOS:=linux}"
: "${REPO_VERSION:=$("${REPO_DIR}/$BUILD_DIR/version")}"

if [ -z "${DOCKER_TAG_SUFFIX:-}" ]; then
  DOCKER_TAG_SUFFIX="$(printf '%s' "${BUILD_TAGS:-}" | tr ' ' '-')"
fi
if [ -z "${DOCKER_TAG:-}" ]; then
  DOCKER_TAG="$(printf '%s' "${REPO_VERSION:-}" | tr ',/' '__')${DOCKER_TAG_SUFFIX:+-}${DOCKER_TAG_SUFFIX:-}"
fi

# Used to stop after a certain FROM stage (target) in the Dockerfile.
if [ -n "${MAKE_DOCKER_TARGET:-}" ]; then
  FULL_DOCKER_TAG="${DOCKER_PROJECT}:target-${MAKE_DOCKER_TARGET}-${DOCKER_TAG}"
else
  FULL_DOCKER_TAG="${DOCKER_PROJECT}:${DOCKER_TAG}"
fi

HEROKU_REGISTRY_DOCKER_TAG="registry.heroku.com/$HEROKU_APP/web"

# Derived Variables: }}}

should_dep_fetch() { [ -z "${NO_DEP:-}" ] && [ -z "${NO_DEP_FETCH:-}" ]; }
# The Make NO_DEP_BUILD was to make build not rely upon the vendored packages,
# which was because I had the binary declared as depending upon the vendored
# paths, which is the right way to do things for Make.
# We don't need it.

#SIDE-EFFECT: sets $FOUND_CMD global (to empty, if not found)
# shellcheck disable=SC2034
have_cmd() {
  local cmd="$1"
  shift
  FOUND_CMD=''
  local oIFS="$IFS"
  IFS=':'
  # Yes we are deliberately doing this to split on whitespace, shellcheck
  # shellcheck disable=SC2086
  set $PATH
  IFS="$oIFS"
  local x
  for x; do
    if [ -x "$x/$cmd" ]; then
      FOUND_CMD="$x/$cmd"
      return 0
    fi
  done
  return 1
}

# XXX old-make-file: call with: $DOCKER_BUILDER_IMAGE $EXTRACT_GO_VERSION_FROM_LABEL
docker_builder_golang_version() {
  local image="${1:?missing docker builder image}"
  local label_with_go_version="${2:?missing docker LABEL which has Go version info}"

  docker pull "${image:?}" >/dev/null
  docker inspect -f "{{index .Config.Labels \"${label_with_go_version}\"}}" "$image"
}

# Support for overriding the Docker ARGs from the build command-line.
# Any environment variable DOCKER_FOO=bar already existing, or any parameter
# env-DOCKER_FOO=bar passed in argv, becomes `--build-arg FOO=bar` to
# override the default value of `ARG FOO=other` in a Dockerfile.
#
# Docker also exposes some ARGs by default, "Predefined ARGs" at
# <https://docs.docker.com/engine/reference/builder/#predefined-args>
# so list them explicitly.
docker_available_ARGs() {
  {
    sed -En 's/^ARG  *([^=]*).*/\1/p' <"$DOCKERFILE"
    printf '%s\n' HTTP_PROXY http_proxy HTTPS_PROXY https_proxy FTP_PROXY ftp_proxy NO_PROXY no_proxy
  } | sort -u
}

go_ldflags_stampversion() {
  printf -- '-X "%s/internal/version.versionString=%s" -X "%s/internal/version.BuildTime=%s"' \
    "$GO_PROJECT_PATH" "$REPO_VERSION" "$GO_PROJECT_PATH" "$BUILD_TIMESTAMP"
}

# find_go_packages is a list of all the non-stdlib packages we depend upon
find_go_packages() {
  "$GO_CMD" list -f '{{join .Imports "\n"}}' ./... |
    sort -u |
    grep -E '^[^/]+\..+/' |
    sed "s:^${GO_PROJECT_PATH}/vendor/::" |
    grep -v "^${GO_PROJECT_PATH}"

}

# The path we place the image in for handoff between docker stages is the
# parent directory of the first directory in $GOPATH.
# For 'docker:N.NN' releases, that means "/$BIN_NAME",
# else probably "$HOME/$BIN_NAME".
binary_handoff_path() {
  printf '%s/%s' "$(dirname "$firstGopath")" "$BIN_NAME"
}

# Call this in CI builds before starting the build, so that we have a report
# of all versions of interest.
# The `dep status` _should_ report everything, but in case it doesn't, we want
# a _thorough_ report, so the `for DIR` line will catch all git repos which we
# depend upon; anything managed by `dep` in `vendor` will be missing a `.git`
# dir and collapse back to the top repo.  Non-git not handled.
show_versions() {
  local real_version DIR

  echo "# Show-versions: {{{"
  date
  uname -a
  "$GIT_CMD" version
  "$GO_CMD" version
  real_version="$("$BUILD_DIR/version")"
  printf 'This repo: %s\n' "$real_version"
  if [ "$real_version" != "$REPO_VERSION" ]; then
    warn "MISMATCH: told via env to build with version: $REPO_VERSION"
  fi
  if have_cmd "$DEP_CMD"; then
    "$DEP_CMD" version
    echo
    "$DEP_CMD" status
  else
    if [ "$DEP_CMD" != "dep" ]; then
      warn "missing '$DEP_CMD' (env replacement of 'dep') command"
    else
      warn "missing '$DEP_CMD' command"
    fi
  fi
  echo
  for DIR in $("$GO_CMD" list -f '{{range .Deps}}{{.}}{{"\n"}}{{end}}' |
    grep -E '^[^/.]+\..*/' |
    xargs "$GO_CMD" list -f '{{.Dir}}' |
    xargs -I {} "$GIT_CMD" -C {} rev-parse --show-toplevel |
    sort -u); do
    printf '%s\t' "$DIR"
    "$GIT_CMD" -C "$DIR" describe --always --dirty --tags
  done
  echo "# Show-versions: }}}"
  echo
}

# This is for _any_ sanity checks before the build, but we'll start with Docker
# tags
prebuild_sanity_check() {
  printf "%s" "$DOCKER_TAG" | grep -qE '^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$'
}

ensure_have_dep() {
  if "$DEP_CMD" version >/dev/null; then return 0; fi
  if [ "$DEP_CMD" != "dep" ]; then
    die "won't auto-install 'dep' when told to use '$DEP_CMD' as dep"
  fi
  if [ "$LOCAL_OS" = "Darwin" ] && [ -d /usr/local/Homebrew ]; then
    brew install dep
    dep version >/dev/null || die "asked brew to install dep, it returned ok, but 'dep' still not in \$PATH"
    return 0
  fi

  # We want the binary stamped with the git information, so that when we ask it
  # a version we get something sane.
  "$GO_CMD" get -u -d github.com/golang/dep/cmd/dep
  local go_hostos go_hostarch
  go_hostos="$("$GO_CMD" env GOHOSTOS)"
  go_hostarch="$("$GO_CMD" env GOHOSTARCH)"
  (
    export DEP_BUILD_PLATFORMS="$go_hostos" DEP_BUILD_ARCHS="$go_hostarch"
    cd "$firstGopath/src/github.com/golang/dep" &&
      ./hack/build-all.bash &&
      install "release/dep-${go_hostos}-${go_hostarch}" "$firstGopath/bin/dep"
  )
  [ -x "$firstGopath/bin/dep" ] || die "installation of dep failed"
  if dep version >/dev/null; then
    return 0
  fi
  DEP_CMD="$firstGopath/bin/dep"
  if "$DEP_CMD" version >/dev/null; then
    return 0
  fi
  die "something is badly wrong, can't run the dep command we just created"
}

have_all_deps() {
  local pkgs p
  pkgs="$(find_go_packages)"
  # shellcheck disable=SC2086
  for p in $pkgs; do
    [ -d "vendor/$p" ] || return 1
  done
  return 0
}

cd "$REPO_DIR"

# vim: set foldmethod=marker :
