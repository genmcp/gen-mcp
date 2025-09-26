CLI_BINARY_NAME = genmcp
SERVER_BINARY_NAME = genmcp-server

.PHONY: clean
clean:
	rm -f $(CLI_BINARY_NAME) $(SERVER_BINARY_NAME)

.PHONY: build-cli
build-cli: clean
	go build -o $(CLI_BINARY_NAME) ./cmd/genmcp

.PHONY: build-server
build-server: clean
	go build -o $(SERVER_BINARY_NAME) ./cmd/genmcp-server

.PHONY: build-all
build-all: clean build-cli build-server
