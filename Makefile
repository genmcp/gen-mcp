CLI_BINARY_NAME = genmcp
SERVER_BINARY_NAME = genmcp-server
BUILD_DIR = pkg/builder/binaries

.PHONY: all
all: build

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(CLI_BINARY_NAME) $(SERVER_BINARY_NAME)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: update-codegen
update-codegen:
	@echo "Running code generation script..."
	@./hack/update-codegen.sh
	@echo "Code generation completed successfully."

.PHONY: build-cli
build-cli: update-codegen
	go build -o $(CLI_BINARY_NAME) ./cmd/genmcp

.PHONY: test
test:
	go test -v -race -count=1 ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build
build: build-cli
