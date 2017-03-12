GOBIN_DIR                   := $(addsuffix /bin, $(shell go env GOPATH))
GOSRC_DIR                   := $(addsuffix /src, $(shell go env GOPATH))
TOP_PKG                      = github.com/operable/circuit-driver
PKG_DIRS                    := $(shell find . -not -path '*/\.*' -type d | grep -v _build | sort)
PKGS                        := $(TOP_PKG) $(subst ., $(TOP_PKG), $(PKG_DIRS))
BUILD_DIR                    = _build
EXE_FILE                    := $(BUILD_DIR)/circuit-driver
DOCKER_IMAGE                ?= "operable/circuit-driver:dev"

# protobuf tooling
PROTOC_BIN                  := $(shell which protoc)
PROTOC_DIR                  := $(dir $(PROTOC_BIN))
PROTO_ROOT                  := $(abspath $(addsuffix .., $(addprefix $(PROTOC_DIR), $(dir $(shell readlink -n $(PROTOC_BIN))))))
PROTO_ROOT_INCLUDE          := $(addsuffix /include/, $(PROTO_ROOT))
GOFAST_PROTOC_BIN           := $(GOBIN_DIR)/protoc-gen-gofast
DRIVER_PROTO_PATH           := $(TOP_PKG)/api
PROTO_INCLUDES              := --proto_path=$(PROTO_ROOT_INCLUDE):$(GOSRC_DIR):$(DRIVER_PROTO_PATH)

.PHONY: all test exe clean docker vet tools pb

all: Makefile test exe

test:
	@go test -cover $(PKGS)

exe: $(BUILD_DIR)
	go build -o $(EXE_FILE) github.com/operable/circuit-driver

vet:
	go $@ $(PKGS)

clean:
	rm -rf $(BUILD_DIR)
	find . -name "*.test" -type f | xargs rm -f

tools: $(GOFAST_PROTOC_BIN)

docker:
	make clean
	GOOS=linux GOARCH=amd64 make exe
	docker build -t $(DOCKER_IMAGE) .

pb-clean:
	rm -f api/*.pb.go

pb:
	cd ../../.. && $(PROTOC_BIN) $(PROTO_INCLUDES) --gofast_out=$(DRIVER_PROTO_PATH) $(DRIVER_PROTO_PATH)/request.proto $(DRIVER_PROTO_PATH)/result.proto

$(GOFAST_PROTOC_BIN):
	go get github.com/gogo/protobuf/protoc-gen-gofast

$(BUILD_DIR):
	mkdir -p $@

