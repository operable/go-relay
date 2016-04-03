GOUPX_BIN          = ../../../../bin/goupx
GOVENDOR_BIN       = ../../../../bin/govendor
GOLINT_BIN         = ../../../../bin/golint
PKG_DIRS          := $(shell find . -type d | grep relay | grep -v vendor)
FULL_PKGS         := $(sort $(foreach pkg, $(PKG_DIRS), $(subst ./, github.com/operable/go-relay/, $(pkg))))
SOURCES           := $(shell find . -name "*.go" -type f)
VET_FLAGS          = -v

ifdef FORCE
.PHONY: all tools lint test clean deps cog-relay
else
.PHONY: all tools lint test clean deps
endif

all: test cog-relay

tools: $(GOUPX_BIN) $(GOVENDOR_BIN) $(GOLINT_BIN)

cog-relay: $(SOURCES) deps
	@rm -f `find . -name "*flymake*.go"`
	go build -o $@ github.com/operable/go-relay

lint: tools
	@for pkg in $(FULL_PKGS); do $(GOLINT_BIN) $$pkg; done

test: tools deps lint
	@go vet $(VET_FLAGS) $(FULL_PKGS)
	@go test -v -cover $(FULL_PKGS)

clean:
	rm -f cog-relay cog-relay-test
	find . -name "*.test" -type f | xargs rm -fv

deps:
	@$(GOVENDOR_BIN) sync
	@go get github.com/fsouza/go-dockerclient

minify: cog-relay
	$(GOUPX_BIN) $<

$(GOUPX_BIN):
	go get -u github.com/pwaller/goupx

$(GOVENDOR_BIN):
	go get -u github.com/kardianos/govendor

$(GOLINT_BIN):
	go get -u github.com/golang/lint/golint
