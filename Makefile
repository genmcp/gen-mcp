CLI_BINARY_NAME = genmcp
SERVER_BINARY_NAME = genmcp-server
BUILD_DIR = pkg/builder/binaries
SERVER_CMD = ./cmd/genmcp-server/

.PHONY: all
all: build-all

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(CLI_BINARY_NAME) $(SERVER_BINARY_NAME)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: build-server-binaries
build-server-binaries: $(BUILD_DIR)
	@echo "Building genmcp-server binaries for all platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/genmcp-server-linux-amd64 $(SERVER_CMD)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/genmcp-server-linux-arm64 $(SERVER_CMD)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/genmcp-server-windows-amd64.exe $(SERVER_CMD)
	@echo "Server binaries built successfully"

.PHONY: build-cli
build-cli: clean build-server-binaries
	go build -o $(CLI_BINARY_NAME) ./cmd/genmcp

.PHONY: build-all
build-all: clean build-server-binaries build-cli
