# This is a GNU makefile
#
# This is also insanely over-engineered and probably buggy, but is designed
# to let most things just happen automatically while still letting lots of things
# be overriden from the command-line or environment.
#
# I believe in automatic tagging of binaries and images with information
# derived from the git repository status, and so forth.  I also play with different
# CI images, set up in various different ways, and want to easily switch with just
# a variable to adapt.
#
# But the core of this, the bit actually needed for building the image, is that
# "build/Dockerfile" # calls "make indocker-build-go" (passing some other vars
# back at the same time, and showing some diagnostic info too).  The
# "indocker-build-go" target invokes "go build"; that could be done directly
# in the Dockerfile, but there's lots of other things we set too.
#
# The Dockerfile is multi-stage, using a builder to make the final tiny image.
# The CI configuration (try .circleci/config.yml) should take care of setting
# up Docker-in-Docker build, to be able to make the Docker image.
#
# IF YOU HAVE FORKED THIS REPO: all the main vars to set to adapt should be
# in the block beginning "GH_PROJECT", just a few lines below here.

.DEFAULT_GOAL := helpful-default
.PHONY: default
default: helpful-default

PRINTABLE_VARS:=

GH_PROJECT    := PennockTech/dummyapp
DOCKERPROJ     = pennocktech/dummyapp
HEROKUAPP     ?= pt-dummy-app
PROJGO        := go.pennock.tech/dummyapp
HEROKU_CR     := registry.heroku.com/$(HEROKUAPP)/web
BINNAME       := dummyapp
CTXPROJDIR    := $(GO_PARENTDIR)go/src/$(PROJGO)
# This depends upon the base docker build image; should end with a /
# The default / is for Golang images, which work in /go
# If building with an image where you are user 'ci' then perhaps: /home/ci/
GO_PARENTDIR  ?= /

PRINTABLE_VARS+= GH_PROJECT DOCKERPROJ HEROKUAPP PROJGO HEROKU_CR BINNAME CTXPROJDIR GO_PARENTDIR

# http://blog.jgc.org/2011/07/gnu-make-recursive-wildcard-function.html
rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))
# mine:
rwildnovendor=$(filter-out vendor/%,$(call rwildcard,$1,$2))

NOOP:=
SPACE:=$(NOOP) $(NOOP)
COMMA:=,

LOCAL_OS    :=$(shell uname)
DOCKER_GOOS ?=linux
GO_CMD      ?=go
GO_LDFLAGS  :=
SOURCES      =$(call rwildnovendor,,*.go)
DEP_VERSION  =$(shell dep version | sed -n 's/^ *version *: *//p')
DOCKERFILE  :=./build/Dockerfile
DEP_FETCH_IN_DOCKER?=
ifdef GOPATH
FIRST_GOPATH:=$(firstword $(subst :,$(SPACE),$(GOPATH)))
else
FIRST_GOPATH:=$(HOME)/go
endif

PRINTABLE_VARS+= LOCAL_OS DOCKER_GOOS GO_CMD GO_LDFLAGS SOURCES DEP_VERSION DOCKERFILE DEP_FETCH_IN_DOCKER FIRST_GOPATH

# This is used for a Docker-in-Docker build approach, where caches are
# optionally loaded in from a cache file; see 'caching-image' target
DIND_CACHE_FILE ?=

# This needs to be within the context passed to the Docker builder, so the
# filesystem can't really be read-only, but it's a bit weird to have to modify
# the source tree on a per-build basis without sub-dirs.  So we support
# moving this and making the parent dir read-only (in theory, not confirmed).
DOCKER_MUTABLE_GO_TAGS ?=build/.docker-go-tags

PRINTABLE_VARS+= DIND_CACHE_FILE DOCKER_MUTABLE_GO_TAGS

# This DOCKER_BUILDER_GOLANG_VERSION target default will fetch Docker images,
# so should only be expanded if absolutely necessary, and then only once.
# Basically: don't set EXTRACT_GO_VERSION_FROM_LABEL unless you really mean it.
ifdef DOCKER_BUILDER_IMAGE
ifdef EXTRACT_GO_VERSION_FROM_LABEL
DOCKER_BUILDER_GOLANG_VERSION := $(shell \
				 docker pull $(DOCKER_BUILDER_IMAGE) >/dev/null && \
				 docker inspect -f '{{index .Config.Labels "$(EXTRACT_GO_VERSION_FROM_LABEL)"}}' $(DOCKER_BUILDER_IMAGE) \
	)
endif
endif

ifdef DEP_FETCH_IN_DOCKER
DOCKER_BUILDER_INSERT_MAKE_TARGETS+= setup
else
DOCKER_NO_DEP_FETCH?=true
build-image: setup
endif

# Support for overriding the Docker ARGs from the Make command-line.
# Any variable DOCKER_FOO to top Make becomes arg FOO for Docker.
# Docker also exposes some ARGs by default, "Predefined ARGs" at
# <https://docs.docker.com/engine/reference/builder/#predefined-args>
# so list them explicitly.  Break the available args out into a variable
# so that print-AVAILABLE_DOCKER_ARGS is usable.
EXTRA_DOCKER_BUILD_ARGS ?=
AVAILABLE_DOCKER_ARGS:=$(shell sed -En 's/^ARG  *([^=]*).*/\1/p' < $(DOCKERFILE) | sort -u) \
	HTTP_PROXY http_proxy HTTPS_PROXY https_proxy FTP_PROXY ftp_proxy NO_PROXY no_proxy
DERIVED_BUILD_ARGS=$(foreach arg,$(AVAILABLE_DOCKER_ARGS),$(if $(DOCKER_${arg}),--build-arg "${arg}=$(DOCKER_${arg})" ,))

PRINTABLE_VARS+= EXTRACT_GO_VERSION_FROM_LABEL EXTRA_DOCKER_BUILD_ARGS $(foreach arg,$(AVAILABLE_DOCKER_ARGS),DOCKER_${arg})
PRINTABLE_VARS+= DERIVED_BUILD_ARGS

# When invoked within Docker, pull in any Go tags which were stashed pre-Docker
ifdef DOCKER_BUILD
ifneq "$(wildcard $(DOCKER_MUTABLE_GO_TAGS) )" ""
BUILD_TAGS ?=$(shell cat $(DOCKER_MUTABLE_GO_TAGS) )
else
BUILD_TAGS ?=
endif
else
BUILD_TAGS ?=
endif
PRINTABLE_VARS+= BUILD_TAGS

ifndef REPO_VERSION
REPO_VERSION :=$(shell ./build/version)
endif
PRINTABLE_VARS+= REPO_VERSION
# POSIX mandates date(1) has `-u` (is the only mandated flag) and mandates
# +format for output per format.
#
# Whether or not to inherit the timestamp is interesting: which is more likely
# to have accurate and trusted time, the container, or the system triggering
# the build in the container?  For now, generate as close to the build as
# possible, ignoring env.
BUILD_TIMESTAMP :=$(shell date -u "+%Y-%m-%d %H:%M:%SZ")
GO_LDFLAGS+= -X "$(PROJGO)/internal/version.VersionString=$(REPO_VERSION)" \
	     -X "$(PROJGO)/internal/version.BuildTime=$(BUILD_TIMESTAMP)"
PRINTABLE_VARS+= BUILD_TIMESTAMP GO_LDFLAGS

ifndef DOCKER_TAG_SUFFIX
ifneq "$(BUILD_TAGS)" ""
DOCKER_TAG_SUFFIX := $(subst $(SPACE),-,$(BUILD_TAGS))
endif
endif

ifndef DOCKER_TAG
ifdef DOCKER_TAG_SUFFIX
# should consider giving build/version a "docker" arg
DOCKER_TAG := $(subst /,_,$(subst $(COMMA),_,$(REPO_VERSION)))-$(DOCKER_TAG_SUFFIX)
else
DOCKER_TAG := $(subst /,_,$(subst $(COMMA),_,$(REPO_VERSION)))
endif
endif
# The docker tags have limits on what is allowed; I've added the
# `prebuild-sanity-check` target for this, and any other such checks before
# build proceeds.
PRINTABLE_VARS+= DOCKER_TAG_SUFFIX DOCKER_TAG

DERIVED_EXTRA_ARGS :=
ifdef MAKE_DOCKER_TARGET
DERIVED_EXTRA_ARGS += --target $(MAKE_DOCKER_TARGET) -t $(DOCKERPROJ):target-$(MAKE_DOCKER_TARGET)-$(DOCKER_TAG)
endif

MAKE_EXTRA_DOCKER_BUILD_ARGS :=$(DERIVED_BUILD_ARGS)$(DERIVED_EXTRA_ARGS) $(EXTRA_DOCKER_BUILD_ARGS) -t $(DOCKERPROJ):$(DOCKER_TAG)


GO_PACKAGES := $(shell $(GO_CMD) list -f '{{join .Imports "\n"}}' ./... | \
	sort -u | \
	egrep '^[^/]+\..+/' | \
	sed 's:^$(PROJGO)/vendor/::' | \
	grep -v '^$(PROJGO)')
VENDORED_GO_PACKAGES := $(foreach p,$(GO_PACKAGES),vendor/$p)
PRINTABLE_VARS+= GO_PACKAGES VENDORED_GO_PACKAGES NO_DEP_BUILD NO_DEP_FETCH NO_DEP

ifdef NO_DEP
NO_DEP_FETCH?=$(NO_DEP)
NO_DEP_BUILD?=$(NO_DEP)
endif

.INTERMEDIATE: setup
setup:

ifndef NO_DEP_FETCH
$(VENDORED_GO_PACKAGES): Gopkg.lock Gopkg.toml have-dep
	dep ensure -v

setup: $(VENDORED_GO_PACKAGES)
endif

ifndef NO_DEP_BUILD
$(BINNAME) $(GO_PARENTDIR)$(BINNAME) : $(VENDORED_GO_PACKAGES)
endif

# build-image boils down to:
#   docker build -t $(DOCKERPROJ) .
# but with a lot of knobs and dials; we make both the untagged implicit-latest
# but also a named versioning tag.
#
.PHONY: build-image
build-image: prebuild-sanity-check
ifneq "$(BUILD_TAGS)" ""
	mkdir -pv "$(shell dirname "$(DOCKER_MUTABLE_GO_TAGS)")"
	printf > $(DOCKER_MUTABLE_GO_TAGS) "%s\n" "$(BUILD_TAGS)"
else
	@rm -f $(DOCKER_MUTABLE_GO_TAGS)
endif
	docker build \
		-t $(DOCKERPROJ) \
		-f $(DOCKERFILE) \
		--build-arg "GO_PARENTDIR=$(GO_PARENTDIR)" \
		--build-arg "APP_VERSION=$(REPO_VERSION)" \
		--build-arg "GO_BUILD_TAGS=$(BUILD_TAGS)" \
		$(MAKE_EXTRA_DOCKER_BUILD_ARGS) \
		.
	@rm -f $(DOCKER_MUTABLE_GO_TAGS)
#		--build-arg "DEP_VERSION=$(DEP_VERSION)" \-eol


.PHONY: push-image
push-image:
	docker push $(DOCKERPROJ):$(DOCKER_TAG)

.PHONY: indocker-build-go
indocker-build-go: prebuild-sanity-check $(GO_PARENTDIR)$(BINNAME)

$(GO_PARENTDIR)$(BINNAME): | prebuild-sanity-check
	cd $(CTXPROJDIR) && \
		CGO_ENABLED=0 GOOS=$(DOCKER_GOOS) \
		$(GO_CMD) build \
		-tags 'docker $(BUILD_TAGS)' \
		-ldflags '$(GO_LDFLAGS) -s' \
		-a -installsuffix docker-nocgo \
		-o $(GO_PARENTDIR)$(BINNAME) \
		$(PROJGO)

# Optionally, rather than multi-stage build Docker, invoked from within Docker,
# we might use an alternative target which depends upon indocker-build-go
# invoked locally, not within Docker.
#
# This would have DOCKERFILE overriden from the command-line to work.
# Assumption: build outside Docker, in appropriate OS, copy files inside
# with a much smaller Dockerfile.
#
# I don't want to maintain a second Dockerfile though, not until I surrender
# and build the things using M4 macros; which, on the bright side, would reduce
# the need for manual ARG duplication.

# You almost certainly want to be using MAKE_DOCKER_TARGET=builder
# with this, and then reinvoking, afterwards with the normal build, because
# otherwise the multi-stage dockerfile will result in caching the tiny final
# image and not all the heavy-weight build-steps needed before that.
#
# TARGET FOR: build-systems
.PHONY: caching-build-image
caching-build-image: step-caching-restore build-image step-caching-persist
#OND:#build-image: | step-caching-restore
step-caching-persist: | step-caching-restore build-image

# The OND: tag means "I suck at Make", I can't figure out a way to declare that
# "If this thing over here is put in the list of hard dependencies by
# something, then it must come before me, but I don't require it for myself".
# "Order, No Dependency".
# Just using `foo: bar baz bat` is not sufficient to force bar to happen before
# baz or bat, with no other constraits upon bar,baz,bat they happened out of
# order.  This is horrendous.

.INTERMEDIATE: step-caching-restore
step-caching-restore:
	@if ! test -n "$(MAKE_DOCKER_TARGET)"; then echo >&2 "Missing: MAKE_DOCKER_TARGET (will cache wrong layer)"; false; fi
	if test -n "$(DIND_CACHE_FILE)" && test -f "$(DIND_CACHE_FILE)"; then \
		ls -ld -- "$(DIND_CACHE_FILE)" ; \
		docker load -i "$(DIND_CACHE_FILE)" -q ; \
	fi

.INTERMEDIATE: step-caching-persist
step-caching-persist:
	if test -n "$(DIND_CACHE_FILE)"; then \
		mkdir -pv "$$(dirname "$(DIND_CACHE_FILE)")" && \
		docker save -o "$(DIND_CACHE_FILE)" $(DOCKERPROJ):target-$(MAKE_DOCKER_TARGET)-$(DOCKER_TAG) ; \
		ls -ld -- "$(DIND_CACHE_FILE)" ; \
	fi

# TARGET FOR: build-systems
.PHONY: persist-build-image
persist-build-image: build-image step-build-image-persist
step-build-image-persist: | build-image

.INTERMEDIATE: step-build-image-persist
step-build-image-persist:
	if test -n "$(DIND_PERSIST_FILE)"; then \
		mkdir -pv "$$(dirname "$(DIND_PERSIST_FILE)")" && \
		docker save -o "$(DIND_PERSIST_FILE)" $(DOCKERPROJ):$(DOCKER_TAG) $(DOCKERPROJ):latest ; \
		ls -ld -- "$(DIND_PERSIST_FILE)" ; \
	fi


LOCALDOCKER_ENVS :=
LOCALDOCKER_ARGS := -log.json
LOCALDOCKER_FLAGS := --rm --read-only -P
PRINTABLE_VARS+= LOCALDOCKER_ENVS LOCALDOCKER_ARGS LOCALDOCKER_FLAGS

# Do manipulation here based on needed env-vars; eg:
#ifeq "$(DATABASE_URL)" ""
#LOCALDOCKER_ARGS += -disable-database
#else
#LOCALDOCKER_ENVS += -e DATABASE_URL="$(DATABASE_URL)"
#endif

LOCALDOCKER_ENVS += -e LOCATION="local-docker on $(shell hostname -s)"

.PHONY: helpful-default
helpful-default: short-help native
#OND:#native: | short-help

.PHONY: localdocker-run
localdocker-run: check-run-env
	id=$$(docker run --detach $(LOCALDOCKER_FLAGS) $(LOCALDOCKER_ENVS) $(DOCKERPROJ):$(DOCKER_TAG) $(LOCALDOCKER_ARGS) ) && \
		echo "Docker ID: $$id" && \
		docker port $$id && \
		docker ps -f id=$$id && \
		if test -n "$(DOCKER_MACHINE_NAME)"; then docker-machine ip $(DOCKER_MACHINE_NAME); fi && \
		docker attach $$id

.PHONY: build-run
build-run: check-run-env build-image localdocker-run
	@true

.PHONY: native-run
native-run: check-run-env native
	./$(BINNAME)

.INTERMEDIATE: native
native: $(BINNAME)

$(BINNAME): setup $(SOURCES) GNUmakefile
	$(GO_CMD) build -o $@ -tags '$(BUILD_TAGS)' -ldflags '$(GO_LDFLAGS)' -v

.INTERMEDIATE: heroku-check
heroku-check:
	@echo Checking for 'heroku' in BUILD_TAGS [$(BUILD_TAGS)]
	@echo "$(BUILD_TAGS)" | xargs -n1 | grep -qs '^heroku$$'

.PHONY: heroku-deploy
heroku-deploy: heroku-check build-image step-heroku-deploy
#OND:#build-image: | heroku-check
# In addition, adding the order dependency here means that when CI invokes
# `make -n step-heroku-deploy`, we print far too much.
#step-heroku-deploy: | heroku-check build-image

.PHONY: step-heroku-deploy
step-heroku-deploy:
	docker tag $(DOCKERPROJ):$(DOCKER_TAG) $(HEROKU_CR)
	docker push $(HEROKU_CR)

.INTERMEDIATE: have-dep
have-dep:
ifeq "$(shell dep version 2>/dev/null)" ""
ifeq ($(LOCAL_OS), Darwin)
ifneq "$(wildcard /usr/local/Homebrew )" ""
	brew install dep
else
	$(GO_CMD) get -u -d github.com/golang/dep/cmd/dep
	# get the git information etc stamped into the binary for version to display when prompted
	hos=$$($(GO_CMD) env GOHOSTOS) har=$$($(GO_CMD) env GOHOSTARCH); cd $(FIRST_GOPATH)/src/github.com/golang/dep && env DEP_BUILD_PLATFORMS=$${hos} DEP_BUILD_ARCHS=$${har} ./hack/build-all.bash && install release/dep-$${hos}-$${har} $(FIRST_GOPATH)/bin/dep
endif
else
	$(GO_CMD) get -u -d github.com/golang/dep/cmd/dep
	hos=$$($(GO_CMD) env GOHOSTOS) har=$$($(GO_CMD) env GOHOSTARCH); cd $(FIRST_GOPATH)/src/github.com/golang/dep && env DEP_BUILD_PLATFORMS=$${hos} DEP_BUILD_ARCHS=$${har} ./hack/build-all.bash && install release/dep-$${hos}-$${har} $(FIRST_GOPATH)/bin/dep
endif
endif

.INTERMEDIATE: check-run-env
check-run-env:
	@true
#ifndef SOME_NEEDED_TOKEN
#	$(error SOME_NEEDED_TOKEN is undefined)
#endif

# This is for _any_ sanity checks before the build, but we'll start with Docker
# tags
.INTERMEDIATE: prebuild-sanity-check
# Docker Tags:
# > A tag name must be valid ASCII and may contain lowercase and uppercase
# > letters, digits, underscores, periods and dashes. A tag name may not start
# > with a period or a dash and may contain a maximum of 128 characters.
prebuild-sanity-check: | setup
	printf "%s" "$(DOCKER_TAG)" | grep -qE '^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$$' # prebuild-sanity-check DOCKER_TAG

# Call this in CI builds before starting the build, so that we have a report
# of all versions of interest.
# The `dep status` _should_ report everything, but in case it doesn't, we want
# a _thorough_ report, so the `for DIR` line will catch all git repos which we
# depend upon; anything managed by `dep` in `vendor` will be missing a `.git`
# dir and collapse back to the top repo.  Non-git not handled.
#
# NB: putting '.git' into .dockerignore would break build/version
.INTERMEDIATE: show-versions
show-versions:
	@echo "# Show-versions:"
	@date
	@uname -a
	@git version
	@$(GO_CMD) version
	@printf "This repo: "; build/version
	if dep version; then dep status; fi
	@echo "Git repo status of repo & dependencies:"
	@for DIR in $$($(GO_CMD) list -f '{{range .Deps}}{{.}}{{"\n"}}{{end}}' | egrep '^[^/.]+\..*/' | xargs $(GO_CMD) list -f '{{.Dir}}' | xargs -I {} git -C {} rev-parse --show-toplevel | sort -u); do echo $$DIR; git -C $$DIR describe --always --dirty --tags ; done
	@echo "# Done with show-versions"

.PHONY: clean
clean:
	$(GO_CMD) clean

.PHONY: short-help
short-help:
	@echo "*** You can try 'make help' for hints on targets ***"
	@echo

.PHONY: help
help:
	@echo "The following targets are available:"
	@echo "  native             build locally, without Docker"
	@echo "  native-run         build locally and run locally"
	@echo "  build-image        build the Docker image"
	@echo "  localdocker-run    use existing Docker build and run there"
	@echo "  build-run          build in Docker and run"
	@echo "  indocker-build-go  intended for use within Docker containers"
	@echo ""
	@echo "  push-image         push image to Docker Hub"
	@echo "  heroku-deploy      build in Docker for heroku and push directly"
	@echo "                     (skipping any CI system normally used)"
	@echo ""
	@echo "  caching-build-image for build-systems, caching intermediates"
	@echo "  persist-build-image for build-systems"
	@echo ""
	@echo "  print-FOO          show the value of the FOO Make variable"
	@echo "  list-vars          list most make variables of interest"
	@echo "  show-vars          show key=value for the variables in list-vars"
	@echo "  show-versions      summary of version numbers of interest"

.INTERMEDIATE: banner-%
banner-%:
	@echo ""
	@echo "*** $* ***"

.PHONY: show-bare-dot-variables
show-bare-dot-variables:
	@env - PATH="$(PATH)" GOPATH="$(GOPATH)" $(MAKE) -rR print-.VARIABLES

.PHONY: list-vars
list-vars:
	@echo >&2 '# This list is incomplete but a rough guide'
	@echo >&2 '# Use print-FOO to see any one variable, here or unlisted'
	@echo >&2 '# Use show-vars to see values for all listed here'
	@echo >&2 '# Try too: make show-bare-dot-variables'
	@printf '%s\n' $(sort $(PRINTABLE_VARS))

# {{{
# Keep these at the end of the file, because that print-% line tends to mess up
# syntax highlighting.

# Where BSD lets you `make -V VARNAME` to print the value of a variable instead
# of building a target, this gives GNU make a target `print-VARNAME` to print
# the value.  I have so missed this when using GNU make.
#
# This rule comes from a comment on
#   <http://blog.jgc.org/2015/04/the-one-line-you-should-add-to-every.html>
# where the commenter provided the shell meta-character-safe version.
.INTERMEDIATE: print-%
print-%: ; @echo '$(subst ','\'',$*=$($*))'

.PHONY: show-vars
show-vars:
	@$(foreach name,$(sort $(PRINTABLE_VARS)),printf '%s=%s\n' '$(name)' '$(subst ','"'"',$($(name)))';)

# }}}
