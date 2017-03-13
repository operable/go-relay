GOLINT_BIN         = $(shell go env GOPATH)/bin/golint
PKG_DIRS          := $(shell find . -type d | grep relay | grep -v vendor)
FULL_PKGS         := $(sort $(foreach pkg, $(PKG_DIRS), $(subst ./, github.com/operable/go-relay/, $(pkg))))
SOURCES           := $(shell find . -name "*.go" -type f)
BUILD_STAMP       := $(shell date -u '+%Y%m%d%H%M%S')
BUILD_HASH        := $(shell git rev-parse HEAD)
BUILD_TAG         ?= $(shell scripts/build_tag.sh)
DRIVER_TAG        ?= 0.13
DOCKER_IMAGE      ?= "operable/relay:$(BUILD_TAG)"
LINK_VARS         := -X main.buildstamp=$(BUILD_STAMP) -X main.buildhash=$(BUILD_HASH)
LINK_VARS         += -X main.buildtag=$(BUILD_TAG) -X main.commanddrivertag=$(DRIVER_TAG)
BUILD_DIR          = _build
EXENAME            = relay

ifdef FORCE
.PHONY: all tools lint test clean deps relay docker
else
.PHONY: all tools lint test clean deps docker
endif

all: test exe

deps:
	govendor sync

vet:
	govendor vet -x +local

test:
	govendor test +local -cover

# This is only intended to run in Travis CI and requires goveralls to
# be installed.
ci-coveralls: tools deps
	goveralls -service=travis-ci

exe: clean-dev | $(BUILD_DIR)
	CGO_ENABLED=0 govendor build -ldflags "$(LINK_VARS)" -o $(BUILD_DIR)/$(EXENAME)

docker:
	make clean
	GOOS=linux GOARCH=amd64 make exe
	make do-docker-build

clean: clean-dev
	rm -rf $(BUILD_DIR) relay-test
	find . -name "*.test" -type f | xargs rm -fv
	find . -name "*-test" -type f | xargs rm -fv

# Remove editor files (here, Emacs)
clean-dev:
	rm -f `find . -name "*flymake*.go"`

$(BUILD_DIR):
	mkdir -p $@

########################################################################
# The targets below stand to be cleaned up. Everything above here is
# analogous to what's in circuit-driver
#

tools: $(GOLINT_BIN)

lint: tools
	@for pkg in $(FULL_PKGS); do $(GOLINT_BIN) $$pkg; done

$(GOLINT_BIN):
	go get -u github.com/golang/lint/golint

tarball: $(TARBALL_NAME)

$(TARBALL_NAME): test exe
	mkdir -p $(TARBALL_NAME)
	cp $(BUILD_DIR)/$(EXENAME) $(TARBALL_NAME)/$(EXENAME)
	cp example_relay.conf $(TARBALL_NAME)
	tar czf $(TARBALL_NAME).tar.gz $(TARBALL_NAME)
	rm -rf $(TARBALL_NAME)

# Providing this solely for CI-built images. We will have already
# built the executable in a separate step. We split things up because
# we build inside a Docker image in CI (we don't have Go on builders).
do-docker-build:
	docker build -t $(DOCKER_IMAGE) .
