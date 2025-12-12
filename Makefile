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

.PHONY: build-cli-platform
build-cli-platform:
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

.PHONY: build-server-platform
build-server-platform:
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then \
		echo "Error: GOOS and GOARCH must be set"; \
		echo "Usage: make build-server-platform GOOS=linux GOARCH=amd64"; \
		exit 1; \
	fi
	@SERVER_NAME="$(SERVER_BINARY_NAME)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		OUTPUT_NAME="$${SERVER_NAME}-$(GOOS)-$(GOARCH).exe"; \
	else \
		OUTPUT_NAME="$${SERVER_NAME}-$(GOOS)-$(GOARCH)"; \
	fi; \
	echo "Building $$OUTPUT_NAME with GOOS=$(GOOS) GOARCH=$(GOARCH)"; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o "$$OUTPUT_NAME" $(SERVER_CMD)

.PHONY: build
build: build-cli

# Extract changelog section for a given version
# Usage: make extract-changelog VERSION=v0.2.0
# For unreleased: make extract-changelog VERSION=Unreleased
.PHONY: extract-changelog
extract-changelog:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION must be set"; \
		echo "Usage: make extract-changelog VERSION=v0.2.0"; \
		echo "       make extract-changelog VERSION=Unreleased"; \
		exit 1; \
	fi
	@sed -n '/## \[$(VERSION)\]/,/## \[/p' CHANGELOG.md | \
		sed '$$d' | \
		tail -n +2 | \
		awk '/^### /{section=$$0; items=""; next} /^( *)?- /{items=items $$0 "\n"; next} /^$$/ && items{print section "\n" items; items=""} END {if (section && items) print section "\n" items}' | \
		sed '/^$$/d'

# Get the latest non-prerelease tag for a given base version (x.y)
# Usage: make latest-release-tag BASE_VERSION=v0.2
# Output: prints the latest release tag (e.g., v0.2.1) or empty if none exists
.PHONY: latest-release-tag
latest-release-tag:
	@if [ -z "$(BASE_VERSION)" ]; then \
		echo "Error: BASE_VERSION must be set" >&2; \
		echo "Usage: make latest-release-tag BASE_VERSION=v0.2" >&2; \
		exit 1; \
	fi; \
	git tag -l "$(BASE_VERSION).*" --sort=-version:refname | grep -v '\-prerelease' | head -n1

# Determine the next release version for a given base version (x.y)
# Finds the latest release tag (excluding prereleases) and increments z
# Usage: make next-release-version BASE_VERSION=v0.2
# Output: prints the next version (e.g., v0.2.0 or v0.2.1)
.PHONY: next-release-version
next-release-version:
	@if [ -z "$(BASE_VERSION)" ]; then \
		echo "Error: BASE_VERSION must be set" >&2; \
		echo "Usage: make next-release-version BASE_VERSION=v0.2" >&2; \
		exit 1; \
	fi; \
	LATEST_RELEASE=$$($(MAKE) -s latest-release-tag BASE_VERSION="$(BASE_VERSION)"); \
	if [ -z "$$LATEST_RELEASE" ]; then \
		echo "$(BASE_VERSION).0"; \
	else \
		LATEST_Z=$$(echo "$$LATEST_RELEASE" | sed "s/$(BASE_VERSION)\.//"); \
		Z_VERSION=$$((LATEST_Z + 1)); \
		echo "$(BASE_VERSION).$$Z_VERSION"; \
	fi
