TOP_PKG                      = github.com/operable/circuit
BUILD_DIR                    = _build
PKG_DIRS                    := $(shell find . -not -path '*/\.*' -type d | grep -v ${BUILD_DIR} | uniq | sort)
PKGS                        := $(subst ., $(TOP_PKG), $(PKG_DIRS))

.PHONY: all test vet

all: vet test

test:
	go test -cover $(PKGS)

vet:
	go vet $(PKGS)
