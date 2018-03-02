# This is a GNU makefile

.DEFAULT_GOAL := helpful-default
.PHONY: default
default: helpful-default

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

# http://blog.jgc.org/2011/07/gnu-make-recursive-wildcard-function.html
rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))
# mine:
rwildnovendor=$(filter-out vendor/%,$(call rwildcard,$1,$2))

LOCAL_OS    :=$(shell uname)
DOCKER_GOOS ?=linux
GO_CMD      ?=go
GO_LDFLAGS  :=
SOURCES      =$(call rwildnovendor,,*.go)
DEP_VERSION  =$(shell dep version | sed -n 's/^ *version *: *//p')
DOCKERFILE  :=./build/Dockerfile

# This is used for a Docker-in-Docker build approach, where caches are
# optionally loaded in from a cache file; see 'caching-image' target
DIND_CACHE_FILE ?=

# This needs to be within the context passed to the Docker builder, so the
# filesystem can't really be read-only, but it's a bit weird to have to modify
# the source tree on a per-build basis without sub-dirs.  So we support
# moving this and making the parent dir.
DOCKER_MUTABLE_GO_TAGS ?=build/.docker-go-tags

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

ifndef REPO_VERSION
REPO_VERSION :=$(shell ./build/version)
endif
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

NOOP:=
SPACE:=$(NOOP) $(NOOP)
COMMA:=,

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

DERIVED_EXTRA_ARGS :=
ifdef MAKE_DOCKER_TARGET
DERIVED_EXTRA_ARGS += --target $(MAKE_DOCKER_TARGET) -t $(DOCKERPROJ):target-$(MAKE_DOCKER_TARGET)-$(DOCKER_TAG)
endif

MAKE_EXTRA_DOCKER_BUILD_ARGS :=$(DERIVED_BUILD_ARGS)$(DERIVED_EXTRA_ARGS) $(EXTRA_DOCKER_BUILD_ARGS) -t $(DOCKERPROJ):$(DOCKER_TAG)

.INTERMEDIATE: setup
setup: have-dep Gopkg.lock
	test -n "$(NODEP)" || dep ensure -v

# build-image boils down to:
#   docker build -t $(DOCKERPROJ) .
# but with a lot of knobs and dials; we make both the untagged implicit-latest
# but also a named versioning tag.
.PHONY: build-image
build-image: setup prebuild-sanity-check
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

$(GO_PARENTDIR)$(BINNAME):
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

# Do manipulation here based on needed env-vars; eg:
#ifeq "$(DATABASE_URL)" ""
#LOCALDOCKER_ARGS += -disable-database
#else
#LOCALDOCKER_ENVS += -e DATABASE_URL="$(DATABASE_URL)"
#endif

LOCALDOCKER_ENVS += -e LOCATION="local-docker on $(shell hostname -s)"

.PHONY: helpful-default
helpful-default: short-help native

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

.PHONY: native
native: setup $(BINNAME)

$(BINNAME): $(SOURCES) GNUmakefile
	$(GO_CMD) build -o $@ -tags '$(BUILD_TAGS)' -ldflags '$(GO_LDFLAGS)' -v

.INTERMEDIATE: heroku-check
heroku-check:
	@echo Checking for 'heroku' in BUILD_TAGS [$(BUILD_TAGS)]
	@echo "$(BUILD_TAGS)" | xargs -n1 | grep -qs '^heroku$$'

.PHONY: heroku-deploy
heroku-deploy: heroku-check build-image step-heroku-deploy

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
	go get -u github.com/golang/dep/cmd/dep
endif
else
	go get -u github.com/golang/dep/cmd/dep
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
prebuild-sanity-check:
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
	@go version
	@printf "This repo: "; build/version
	if dep version; then dep status; fi
	@echo "Git repo status of repo & dependencies:"
	@for DIR in $$(go list -f '{{range .Deps}}{{.}}{{"\n"}}{{end}}' | egrep '^[^/.]+\..*/' | xargs go list -f '{{.Dir}}' | xargs -I {} git -C {} rev-parse --show-toplevel | sort -u); do echo $$DIR; git -C $$DIR describe --always --dirty --tags ; done
	@echo "# Done with show-versions"

.PHONY: clean
clean:
	go clean

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
	@echo "  show-versions      summary of version numbers of interest"

.INTERMEDIATE: banner-%
banner-%:
	@echo ""
	@echo "*** $* ***"

# Where BSD lets you `make -V VARNAME` to print the value of a variable instead
# of building a target, this gives GNU make a target `print-VARNAME` to print
# the value.  I have so missed this when using GNU make.
#
# This rule comes from a comment on
#   <http://blog.jgc.org/2015/04/the-one-line-you-should-add-to-every.html>
# where the commenter provided the shell meta-character-safe version.
.PHONY: print-%
print-%: ; @echo '$(subst ','\'',$*=$($*))'
# Keep this at the end of the file, because that print-% line tends to mess up
# syntax highlighting.
