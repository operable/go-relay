GOUPX_BIN          = ../../../../bin/goupx
GOVENDOR_BIN       = ../../../../bin/govendor
GOLINT_BIN         = ../../../../bin/golint
PKG_DIRS          := $(shell find . -type d | grep relay | grep -v vendor)
PACKAGES          := $(sort $(foreach pkg, $(PKG_DIRS), $(subst ./, github.com/operable/go-relay/, $(pkg))))
SOURCES           := $(shell find . -name "*.go" -type f)
VET_FLAGS          = -v

ifdef FORCE
.PHONY: all tools test clean deps cog-relay
else
.PHONY: all tools test clean deps
endif

all: test cog-relay

tools: $(GOUPX_BIN) $(GOVENDOR_BIN) $(GOLINT_BIN)

cog-relay: $(SOURCES) deps
	@rm -f `find . -name "*flymake*.go"`
	go build -o $@ github.com/operable/go-relay

test: tools deps
	@golint github.com/operable/go-relay/relay
	@go vet $(VET_FLAGS) $(PACKAGES)
	@go test -v -cover $(PACKAGES)

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
