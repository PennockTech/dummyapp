# Dockerfile for: dummyapp
# This is a multi-stage Dockerfile: a builder and and then a final image
# creator.  Many parameters can be overriden with Docker build-args.

# Each ARG goes out of scope immediately after a FROM line, so repeat
# in each stage where needed.  If changing these, then check at the end
# of the file where we apply labels to the final image.

# First, we build; we make it easy to just change the version of Go, but
# you can use BUILDER_IMAGE to just use a completely different builder.
# If you do so, please try to at least also pass BUILDER_GOLANG_VERSION
# to be accurate, as this information is labelled into the final built
# image.
ARG BUILDER_GOLANG_VERSION=1.10
ARG BUILDER_VARIANT=""
ARG BUILDER_IMAGE_BASE="golang"
ARG BUILDER_IMAGE="${BUILDER_IMAGE_BASE}:${BUILDER_GOLANG_VERSION}${BUILDER_VARIANT}"
FROM ${BUILDER_IMAGE} AS builder

# This is used to have Docker's invocation of make do extra steps before
# the build; it is normally empty.
ARG BUILDER_INSERT_MAKE_TARGETS=""
# This is used when the build image has $GOPATH as somewhere other than /go
# and should then be, for builds as user "ci" in ~ci/go: /home/ci/
# (assuming normal modern Unix homedir conventions).
ARG GO_PARENTDIR="/"

# Our build-context should be a the repo itself.
#
# We exclude the vendor area from the git repo, but expect it to have
# been populated before Docker is called.
# If you want dependencies fetched and built within Docker instead of in
# the parent, then remove 'setup' from the dependencies for make's 'build-image'
# and pass it in via BUILDER_INSERT_MAKE_TARGETS.
#
ADD . ${GO_PARENTDIR}go/src/go.pennock.tech/dummyapp
WORKDIR ${GO_PARENTDIR}go/src/go.pennock.tech/dummyapp
ENV DOCKER_BUILD=true
RUN make -w \
	GO_PARENTDIR=${GO_PARENTDIR} \
	${BUILDER_INSERT_MAKE_TARGETS} \
	show-versions banner-start-build \
	indocker-build-go

# ======================8< New Stage: Final Image >8======================

# If changing this, be sure to update the label too
ARG RUNTIME_BASE_IMAGE="scratch"
FROM ${RUNTIME_BASE_IMAGE}
ARG RUNTIME_BASE_IMAGE="scratch"
ARG PORT="8080"
ARG GO_PARENTDIR="/"

# Warning: changing this doesn't currently affect the build, only what's
# put into the image label.  Unfortunately, COPY --from does not interpolate
# variables for the source image name.
ARG POETRY_IMAGE=philpennock/poetry

EXPOSE ${PORT}

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
ENTRYPOINT ["/dummyapp"]
CMD ["-log.json"]

# The program itself
COPY --from=builder ${GO_PARENTDIR}dummyapp /

ENV PORT=${PORT}

# And the data-files, which might normally be assumed to vary often
#want: COPY --from=${POETRY_IMAGE} /* /poetry/
# Okay, what we really want is conditional directives in the Dockerfile, so we
# can do "if POETRY_IMAGE is defined, then copy from it and set this env var".
COPY --from=philpennock/poetry /* /poetry/

ENV POETRY_DIR=/poetry

# These are replicated ARGs for labelling into the final image
# Layers introduced by LABEL coalesce into one.
ARG BUILDER_GOLANG_VERSION=1.10
ARG BUILDER_VARIANT=""
ARG BUILDER_IMAGE_BASE="golang"
ARG BUILDER_IMAGE="${BUILDER_IMAGE_BASE}:${BUILDER_GOLANG_VERSION}${BUILDER_VARIANT}"
ARG BUILDER_INSERT_MAKE_TARGETS=""
ARG APP_VERSION="<unknown>"
ARG DEP_VERSION="<unknown>"
ARG GO_BUILD_TAGS="<unknown>"
LABEL maintainer="noc+di@pennock-tech.com"
LABEL tech.pennock.name="dummyapp"
LABEL tech.pennock.builder.image="${BUILDER_IMAGE}"
LABEL tech.pennock.builder.insertedmake="${BUILDER_INSERT_MAKE_TARGETS}"
LABEL tech.pennock.baseimage="${RUNTIME_BASE_IMAGE}"
LABEL tech.pennock.portlist="${PORT}"
LABEL tech.pennock.version="${APP_VERSION}"
LABEL tech.pennock.versions.go="${BUILDER_GOLANG_VERSION}"
LABEL tech.pennock.versions.dep="${DEP_VERSION}"
LABEL tech.pennock.gobuildtags="${GO_BUILD_TAGS}"
LABEL tech.pennock.data.poetryimage="${POETRY_IMAGE}"
# If we build dep into our own image, we'd tag that in the base.
