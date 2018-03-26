# Copyright © 2018 Pennock Tech, LLC.
# All rights reserved, except as granted under license.
# Licensed per file LICENSE.txt

# Dockerfile for: dummyapp
# This is a multi-stage Dockerfile: a builder and and then a final image
# creator.  Many parameters can be overriden with Docker build-args.

# Each ARG goes out of scope immediately after a FROM line, so repeat
# in each stage where needed.  Be careful here.
# The default value does _not_ need to be repeated, but *only* for ARGs
# which are declared before the first FROM.
# Also, "FROM instructions support variables that are declared by any ARG
# instructions that occur before the first FROM"
# So pre-declare them *all*.
#
# If each stage were FROM the previous stage, we could just LABEL in the
# stage which needs the ARG and it would carry across; but because we
# use `FROM scratch`, we lose all that.

# First, we build; we make it easy to just change the version of Go, but
# you can use BUILDER_IMAGE to just use a completely different builder.
# If you do so, please try to at least also pass BUILDER_GOLANG_VERSION
# to be accurate, as this information is labelled into the final built
# image.
ARG BUILDER_GOLANG_VERSION=1.10
ARG BUILDER_VARIANT=""
ARG BUILDER_IMAGE_BASE="golang"
ARG BUILDER_IMAGE="${BUILDER_IMAGE_BASE}:${BUILDER_GOLANG_VERSION}${BUILDER_VARIANT}"

# These are all documented inline below, where actually used
ARG GO_PARENTDIR="/"
ARG GO_BUILD_TAGS=""
ARG NO_DEP_FETCH=""
ARG POETRY_DIR=/poetry
ARG POETRY_IMAGE="philpennock/poetry"
ARG PORT="8080"
ARG RUNTIME_BASE_IMAGE="scratch"

# ========================8< New Stage: Builder >8========================
FROM ${BUILDER_IMAGE} AS builder

# This is used when the build image has $GOPATH as somewhere other than /go
# and should then be, for builds as user "ci" in ~ci/go: /home/ci/
# (assuming normal modern Unix homedir conventions).
ARG GO_PARENTDIR
# This is used to prevent having dep inside Docker try to fetch dependencies
# if it thinks things are out of date:
ARG NO_DEP_FETCH
# This is used to propagate Golang build tags from the outside, through Docker,
# into the actual Go build used.  Note that the actual build, per normal, uses
# BUILD_TAGS not GO_BUILD_TAGS, so we rename back.  It's only renamed to have
# the GO_ prefix when invoking Docker, and that's to make it clear on inspection
# that it relates to the Go build, not the Docker build.
ARG GO_BUILD_TAGS

# Our build-context should be a the repo itself.
#
# We exclude the vendor area from the git repo, but expect it to have
# been populated before Docker is called.
#
# If you want dependencies fetched and built within Docker instead of in
# the parent, then build with DEP_FETCH_IN_DOCKER=t.

ADD . ${GO_PARENTDIR}go/src/go.pennock.tech/dummyapp
WORKDIR ${GO_PARENTDIR}go/src/go.pennock.tech/dummyapp
ENV DOCKER_BUILD=true
RUN build/build.from-within-dockerfile.sh env-GO_PARENTDIR="${GO_PARENTDIR}" env-NO_DEP_FETCH="${NO_DEP_FETCH}" env-BUILD_TAGS="${GO_BUILD_TAGS}"

# =====================8< New Stage: Poetry Image >8======================
# Work-around for being unable to specify an ARG as `COPY --from=` source.
# Per @tonistiigi in
#   <https://github.com/moby/moby/issues/34482#issuecomment-332298635>
# The rationale for the restriction makes sense: want to know the complete
# list of dependencies before pulling the images, and everything other than
# FROM which interpolates, interpolates ENV+ARG, not ARG-only.  Rather than
# add more variance, just pull the image.

ARG POETRY_IMAGE
FROM ${POETRY_IMAGE} AS poetry

# ======================8< New Stage: Final Image >8======================

ARG RUNTIME_BASE_IMAGE
FROM ${RUNTIME_BASE_IMAGE}
ARG RUNTIME_BASE_IMAGE
ARG PORT
ARG GO_PARENTDIR
ARG POETRY_DIR

# When adding the storage layers, add the rarest-changing one first, so that
# content-addressing might better support layer reuse.
# _Then_ add the actual binary.

# Data files loaded by the Go stdlib
COPY build/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# TODO: timezone files, for Go 1.10

# Putting an ${ARG} into CMD forces shell, there's no way to have a const number
# baked in, that I can tell.  So we switched to ENV for ${PORT}.
#
# Heroku ignores that array-form CMD is supposed to be shell-less and tries to
# use shell anyway, so the only way around it is to use entrypoint instead.
#ENTRYPOINT ["/dummyapp"]
CMD ["/dummyapp", "-log.json"]

# The program itself
COPY --from=builder ${GO_PARENTDIR}dummyapp /
# And the data files
COPY --from=poetry / ${POETRY_DIR}/

# Let the poetry be overriden at run-time if the Container hosting environment
# supports Volumes.  Note that Heroku does not, per:
#   <https://devcenter.heroku.com/articles/container-registry-and-runtime>
#
# Specifying this after the COPY should default the volume contents to
# be those we've pre-populated.
VOLUME ${POETRY_DIR}

EXPOSE ${PORT}

# Expose environment variables to the app, which accepts these as
# defaults.
ENV PORT=${PORT}
ENV POETRY_DIR=${POETRY_DIR}

# These are replicated ARGs for labelling into the final image
# Layers introduced by LABEL coalesce into one.
ARG BUILDER_GOLANG_VERSION
ARG BUILDER_VARIANT
ARG BUILDER_IMAGE_BASE
ARG BUILDER_IMAGE
ARG POETRY_IMAGE
ARG GO_BUILD_TAGS
# These are only for the labels and not repeats:
ARG APP_VERSION="<unknown>"
ARG DEP_VERSION="<unknown>"
#
LABEL maintainer="noc+di@pennock-tech.com"
LABEL tech.pennock.name="dummyapp"
LABEL tech.pennock.builder.image="${BUILDER_IMAGE}"
LABEL tech.pennock.baseimage="${RUNTIME_BASE_IMAGE}"
LABEL tech.pennock.portlist="${PORT}"
LABEL tech.pennock.version="${APP_VERSION}"
LABEL tech.pennock.versions.go="${BUILDER_GOLANG_VERSION}"
LABEL tech.pennock.versions.dep="${DEP_VERSION}"
LABEL tech.pennock.gobuildtags="${GO_BUILD_TAGS}"
LABEL tech.pennock.data.poetryimage="${POETRY_IMAGE}"
LABEL tech.pennock.volumes.poetry="${POETRY_DIR}"
# If we build dep into our own image, we'd tag that in the base.
#
# The value from `docker inspect <image> | .[0].Created` field is "image
# recorded as the actual build-date", not "image created in this Docker
# system".  I know this because the gcr.io/distroless images are built with a
# zeroed timestamp and show up as "48 years ago" and the same jq invocation
# shows that's what's stamped in them.
# So we don't try to reproduce that timestamp in our LABELs.
#
# Arguably, we should also avoid repeating the VOLUME specifiers, but the
# volumes are keyed by path, not purpose; our label is keyed by purpose.
# But then, if the paths are so unclear that the purpose isn't obvious,
# then perhaps that should be fixed?  After all, we run isolated and can
# put what we want, where we want.
