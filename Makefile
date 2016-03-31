GOUPX_BIN          = ../../../../bin/goupx
GOVENDOR_BIN       = ../../../../bin/govendor
GOLINT_BIN         = ../../../../bin/golint

ifdef FORCE
.PHONY: all tools test clean cog-relay
else
.PHONY: all tools test clean
endif

all: test

tools: $(GOUPX_BIN) $(GOVENDOR_BIN) $(GOLINT_BIN)

cog-relay:
	@rm -f `find . -name "*flymake*.go"`
	@$(GOVENDOR_BIN) sync
	@go get github.com/fsouza/go-dockerclient
	go build -o $@ github.com/operable/go-relay

test: tools cog-relay
	golint github.com/operable/go-relay/relay
	go test -cover -v -o cog-relay-test github.com/operable/go-relay/relay

clean:
	rm -f cog-relay cog-relay-test

minify:
	$(GOUPX_BIN) cog-relay

$(GOUPX_BIN):
	go get -u github.com/pwaller/goupx

$(GOVENDOR_BIN):
	go get -u github.com/kardianos/govendor

$(GOLINT_BIN):
	go get -u github.com/golang/lint/golint
