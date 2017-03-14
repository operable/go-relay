EXE                = cogexec
BUILD_DIR          = _build

.PHONY: all clean docker

all: $(BUILD_DIR)/$(EXE)

clean:
	rm -rf $(BUILD_DIR)

docker: clean
	GOOS=linux GOARCH=amd64 make all
	docker build -t operable/cogexec .

$(BUILD_DIR)/$(EXE): $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(EXE) github.com/operable/cogexec

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)
