GOVENDOR_BIN       = $(shell go env GOPATH)/bin/govendor
GOLINT_BIN         = $(shell go env GOPATH)/bin/golint
PKG_DIRS          := $(shell find . -type d | grep relay | grep -v vendor)
FULL_PKGS         := $(sort $(foreach pkg, $(PKG_DIRS), $(subst ./, github.com/operable/go-relay/, $(pkg))))
SOURCES           := $(shell find . -name "*.go" -type f)
VET_FLAGS          = -v
BUILD_STAMP       := $(shell date -u '+%Y%m%d%H%M%S')
BUILD_HASH        := $(shell git rev-parse HEAD)
BUILD_TAG         ?= $(shell git describe --tags)
DOCKER_IMAGE      ?= "operable/relay:0.7-dev"
LINK_VARS         := -X main.buildstamp=$(BUILD_STAMP) -X main.buildhash=$(BUILD_HASH)
LINK_VARS         += -X main.buildtag=$(BUILD_TAG)
OSNAME            := $(shell uname | tr A-Z a-z)
ARCHNAME          := $(shell $(CC) -dumpmachine | cut -d\- -f1)
BUILD_DIR          = _build
EXENAME            = relay

ifeq ($(ARCHNAME), x86_64)
ARCHNAME           = amd64
endif
TARBALL_NAME       = $(EXENAME)_$(OSNAME)_$(ARCHNAME)


ifdef FORCE
.PHONY: all tools lint test clean deps relay docker
else
.PHONY: all tools lint test clean deps docker
endif

all: test exe

exe: $(BUILD_DIR)/$(EXENAME)

tools: $(GOVENDOR_BIN) $(GOLINT_BIN)

$(BUILD_DIR)/$(EXENAME): $(BUILD_DIR) $(SOURCES) tools deps
	@rm -f `find . -name "*flymake*.go"`
	@rm -rf relay_*_amd64
	go build -ldflags "$(LINK_VARS)" -o $@ github.com/operable/go-relay

lint: tools
	@for pkg in $(FULL_PKGS); do $(GOLINT_BIN) $$pkg; done

test: tools deps lint
	@rm -rf relay_*_amd64
	@go vet $(VET_FLAGS) $(FULL_PKGS)
	@go test -v -cover $(FULL_PKGS)

clean:
	rm -rf $(BUILD_DIR) relay-test
	find . -name "*.test" -type f | xargs rm -fv
	find . -name "*-test" -type f | xargs rm -fv

deps:
	@$(GOVENDOR_BIN) sync
	@go get -u github.com/fsouza/go-dockerclient

$(GOVENDOR_BIN):
	go get -u github.com/kardianos/govendor

$(GOLINT_BIN):
	go get -u github.com/golang/lint/golint

tarball: $(TARBALL_NAME)

$(TARBALL_NAME): test exe
	mkdir -p $(TARBALL_NAME)
	cp $(BUILD_DIR)/$(EXENAME) $(TARBALL_NAME)/$(EXENAME)
	cp example_relay.conf $(TARBALL_NAME)
	tar czf $(TARBALL_NAME).tar.gz $(TARBALL_NAME)
	rm -rf $(TARBALL_NAME)

docker: clean
	GOOS=linux GOARCH=amd64 make exe
	docker build -t $(DOCKER_IMAGE) .

$(BUILD_DIR):
	mkdir -p $@
