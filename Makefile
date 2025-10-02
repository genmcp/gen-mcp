CLI_BINARY_NAME = genmcp
SERVER_BINARY_NAME = genmcp-server
BUILD_DIR = pkg/builder/binaries
SERVER_CMD = ./cmd/genmcp-server/

.PHONY: all
all: build

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(CLI_BINARY_NAME) $(SERVER_BINARY_NAME)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: build-server-binaries
build-server-binaries: clean $(BUILD_DIR)
	@echo "Building genmcp-server binaries for all platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/genmcp-server-linux-amd64 $(SERVER_CMD)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/genmcp-server-linux-arm64 $(SERVER_CMD)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/genmcp-server-windows-amd64.exe $(SERVER_CMD)
	@echo "Server binaries built successfully"

.PHONY: build-cli
build-cli: build-server-binaries
	go build -o $(CLI_BINARY_NAME) ./cmd/genmcp

.PHONY: test
test: build-server-binaries
	go test -v -race -count=1 ./...

.PHONY: lint
lint: build-server-binaries
	golangci-lint run

.PHONY: build-cli-platform
build-cli-platform: build-server-binaries
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then \
		echo "Error: GOOS and GOARCH must be set"; \
		echo "Usage: make build-cli-platform GOOS=linux GOARCH=amd64 [VERSION_TAG=v1.0.0]"; \
		exit 1; \
	fi
	@CLI_NAME="$(CLI_BINARY_NAME)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		OUTPUT_NAME="$${CLI_NAME}-$(GOOS)-$(GOARCH).exe"; \
	else \
		OUTPUT_NAME="$${CLI_NAME}-$(GOOS)-$(GOARCH)"; \
	fi; \
	echo "Building $$OUTPUT_NAME with GOOS=$(GOOS) GOARCH=$(GOARCH)"; \
	if [ -n "$(VERSION_TAG)" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-X main.version=$(VERSION_TAG)" -o "$$OUTPUT_NAME" ./cmd/genmcp; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o "$$OUTPUT_NAME" ./cmd/genmcp; \
	fi

.PHONY: build
build: build-server-binaries build-cli
