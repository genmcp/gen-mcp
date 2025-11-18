---
layout: page
title: Command Reference
description: Complete reference for all gen-mcp CLI commands
---

# Command Reference

The `genmcp` CLI provides commands for managing MCP servers, converting API specifications, and building container images. This guide covers all available commands with detailed explanations and examples.

---

## Quick Reference

| Command | Description | Common Usage |
|---------|-------------|--------------|
| [`run`](#run) | Start an MCP server | `genmcp run -t mcpfile.yaml -s mcpserver.yaml` |
| [`stop`](#stop) | Stop a running server | `genmcp stop -f mcpfile.yaml` |
| [`convert`](#convert) | Convert OpenAPI to MCP | `genmcp convert openapi.json` |
| [`build`](#build) | Build container image | `genmcp build -f mcpfile.yaml -s mcpserver.yaml --tag myapi:latest` |
| [`version`](#version) | Display version info | `genmcp version` |

---

## <span style="color: #E6622A;">run</span>

Start an MCP server from an MCP file configuration.

#### Usage

```bash
genmcp run [flags]
```

#### Flags

| Flag              | Short | Default          | Description                                            |
|-------------------|-------|------------------|--------------------------------------------------------|
| `--file`          | `-f`  | `mcpfile.yaml`   | Path to the tool definitions file (MCPToolDefinitions) |
| `--server-config` | `-s`  | `mcpserver.yaml` | Path to the server config file (MCPServerConfig)       |
| `--detach`        | `-d`  | `false`          | Run server in background (detached mode)               |

#### How It Works

The `run` command:

1. **Validates both files** - Checks syntax and schema validity of both the tool definitions and server config files
2. **Validates invocations** - Ensures all tool invocations are properly configured
3. **Starts the server** - Launches the MCP server with the specified transport protocol
4. **Manages lifecycle** - In detached mode, saves the process ID for later management

**Transport Protocols:**
- `stdio` - Standard input/output (for local integrations like Claude Desktop)
- `streamablehttp` - HTTP-based server (for network-accessible integrations)

#### Examples

**Basic usage:**
```bash
# Run with default files (mcpfile.yaml and mcpserver.yaml)
genmcp run

# Run with specific files
genmcp run -t ./config/tools.yaml -s ./config/server.yaml

# Run with absolute paths
genmcp run -t /path/to/mcpfile.yaml -s /path/to/mcpserver.yaml
```

**Detached mode (background):**
```bash
# Start server in background
genmcp run -t mcpfile.yaml -s mcpserver.yaml --detach

# Server runs independently, can close terminal
# Use 'genmcp stop' to terminate later
```

**Real-world scenarios:**

```bash
# Development: Run in foreground with logs visible
cd examples/ollama
genmcp run -t ollama-http.yaml -s ollama-mcpserver.yaml

# Production: Run in background
genmcp run -t /etc/genmcp/tools.yaml -s /etc/genmcp/server.yaml -d

# Testing: Quick validation and startup
genmcp run -t test-tools.yaml -s test-server.yaml
# Press Ctrl+C to stop when done testing
```

#### Notes

- **Two files required**: Both tool definitions and server config files must be provided
- **Detached mode with stdio**: The `--detach` flag is automatically disabled when using `stdio` transport protocol, as stdio requires continuous process connection
- **Validation errors**: The command will fail fast if either file has syntax errors or invalid configurations
- **Process management**: When running in detached mode, the process ID is saved to allow the `stop` command to terminate the server

---

## <span style="color: #E6622A;">stop</span>

Stop a running MCP server that was started in detached mode.

#### Usage

```bash
genmcp stop [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `mcpfile.yaml` | Path to the tool definitions file of the server to stop (used as identifier) |

#### How It Works

The `stop` command:

1. **Resolves the tool definitions file path** - Finds the absolute path to match the running server (uses tool definitions file as identifier)
2. **Retrieves the process ID** - Looks up the saved PID from when the server was started
3. **Terminates the process** - Sends a kill signal to stop the server
4. **Cleans up** - Removes the saved process ID

#### Examples

**Basic usage:**
```bash
# Stop server using default mcpfile.yaml (tool definitions file)
genmcp stop

# Stop server with specific tool definitions file
genmcp stop -f ollama-http.yaml

# Stop server with absolute path
genmcp stop -f /path/to/mcpfile.yaml
```

**Workflow example:**
```bash
# Start server in background
genmcp run -t myapi.yaml -s myapi-server.yaml --detach
# Output: successfully started gen-mcp server...

# Later, stop the server (use tool definitions file path)
genmcp stop -f myapi.yaml
# Output: successfully stopped gen-mcp server...
```

#### Notes

- **File path must match**: The tool definitions file path used with `stop` must match the path used with `run --detach`
- **Only works with detached servers**: Servers running in foreground mode can be stopped with Ctrl+C
- **Manual cleanup**: If the process was manually killed outside of gen-mcp, you may need to manually clean up the saved PID file

---

## <span style="color: #E6622A;">convert</span>

Convert an OpenAPI v2 or v3 specification into an MCP file configuration.

#### Usage

```bash
genmcp convert <openapi-spec> [flags]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `<openapi-spec>` | URL or file path to OpenAPI specification (JSON or YAML) |

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--out` | `-o` | `mcpfile.yaml` | Output path for the generated tool definitions file |
| `--host` | `-H` | *(from spec)* | Override the base host URL from the OpenAPI spec |

#### How It Works

The `convert` command:

1. **Fetches the spec** - Downloads from URL or reads from file
2. **Parses OpenAPI** - Supports both OpenAPI v2 (Swagger) and v3 formats
3. **Generates tools** - Creates an MCP tool for each API endpoint
4. **Maps schemas** - Converts OpenAPI parameter schemas to JSON Schema for input validation
5. **Creates invocations** - Generates HTTP invocations with proper methods and URLs
6. **Writes MCP files** - Outputs both a tool definitions file and a server config file

#### Examples

**Convert from URL:**
```bash
# Public API
genmcp convert https://petstore.swagger.io/v2/swagger.json

# Local server
genmcp convert http://localhost:8080/openapi.json

# With custom output path
genmcp convert https://api.example.com/openapi.yaml -o my-api.yaml
```

**Convert from file:**
```bash
# Local OpenAPI file
genmcp convert ./api-spec.json

# With custom output location
genmcp convert ./specs/v3-api.yaml -o ./configs/mcp-api.yaml
```

**Override host URL:**
```bash
# Original spec has https://api.example.com
# Override to use local dev server
genmcp convert openapi.json --host http://localhost:3000

# Override to use staging environment
genmcp convert openapi.json -H https://staging-api.example.com -o staging.yaml
```

**Complete workflow:**
```bash
# 1. Convert OpenAPI spec (generates both files)
genmcp convert https://api.github.com/openapi.json -o github-tools.yaml
# Output: wrote tool definitions to github-tools.yaml
# Output: wrote server config to github-server.yaml

# 2. Review and customize the generated files
cat github-tools.yaml
cat github-server.yaml
# Edit descriptions, add safety guards, etc.

# 3. Run the MCP server
genmcp run -t github-tools.yaml -s github-server.yaml
```

#### Generated Structure

The converter automatically creates two files:

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: API Name                    # From OpenAPI info.title
version: 1.0.0                    # From OpenAPI info.version

invocationBases:
  baseApi:
    http:
      url: https://api.example.com  # From OpenAPI servers

tools:
- name: get_users                 # Generated from operationId or path
  title: Get Users                # From OpenAPI summary
  description: "..."              # From OpenAPI description
  inputSchema:                    # From OpenAPI parameters
    type: object
    properties: { ... }
  invocation:
    extends:
      from: baseApi
      extend:
        url: /users               # From OpenAPI path
      override:
        method: GET               # From OpenAPI method
```

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
name: API Name                    # From OpenAPI info.title
version: 1.0.0                    # From OpenAPI info.version
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
```

#### Customizing Generated Files

After conversion, you'll typically want to:

1. **Improve descriptions** - Add context for LLM tool selection
2. **Add safety guards** - Warn about destructive operations
3. **Adjust invocation bases** - Group related endpoints
4. **Refine schemas** - Add validation rules or constraints

See the [HTTP API Conversion Example]({{ '/example-http-conversion.html' | relative_url }}) for a detailed walkthrough.

---

## <span style="color: #E6622A;">build</span>

Build a container image containing your MCP server and configuration.

#### Usage

```bash
genmcp build [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `mcpfile.yaml` | Path to tool definitions file to include in image |
| `--server-config` | `-s` | `mcpserver.yaml` | Path to server config file to include in image |
| `--tag` | | *(required)* | Image tag (e.g., `myregistry/myapi:v1.0`) |
| `--base-image` | | *(auto)* | Base container image to build on |
| `--platform` | | `multi-arch` | Target platform (e.g., `linux/amd64`) |
| `--push` | | `false` | Push to registry instead of saving locally |

#### How It Works

The `build` command:

1. **Validates both MCP files** - Ensures both tool definitions and server config are valid
2. **Builds container image** - Creates a containerized MCP server with both files included
3. **Supports multi-arch** - By default builds for `linux/amd64` and `linux/arm64`
4. **Saves or pushes** - Either stores locally or pushes to a container registry

#### Examples

**Basic local build:**
```bash
# Build and save to local Docker daemon (uses default files)
genmcp build --tag myapi:latest

# Build with specific files
genmcp build -f config/tools.yaml -s config/server.yaml --tag myapi:v1.0

# Build for specific platform only
genmcp build --tag myapi:latest --platform linux/amd64
```

**Multi-architecture build:**
```bash
# Default: builds for both amd64 and arm64
genmcp build --tag myregistry/myapi:v1.0

# Creates platform-specific tags locally:
# - myregistry/myapi:v1.0-linux-amd64
# - myregistry/myapi:v1.0-linux-arm64
```

**Push to registry:**
```bash
# Build and push to Docker Hub
genmcp build --tag username/myapi:v1.0 --push

# Build and push to private registry
genmcp build --tag registry.company.com/myapi:latest --push

# Note: Ensure you're logged in first
# docker login registry.company.com
```

**Custom base image:**
```bash
# Use specific base image
genmcp build --tag myapi:latest --base-image alpine:latest

# Use distroless for minimal image
genmcp build --tag myapi:latest --base-image gcr.io/distroless/base
```

**Production workflow:**
```bash
# 1. Build multi-arch image
genmcp build \
  -f production-tools.yaml \
  -s production-server.yaml \
  --tag myregistry.io/production-api:v2.1.0 \
  --push

# 2. Deploy to Kubernetes
kubectl set image deployment/mcp-server \
  mcp=myregistry.io/production-api:v2.1.0

# 3. Tag as latest if successful
docker tag myregistry.io/production-api:v2.1.0 myregistry.io/production-api:latest
docker push myregistry.io/production-api:latest
```

#### Notes

- **Registry authentication**: When using `--push`, ensure you're authenticated with the target registry
- **Multi-arch builds**: Without `--platform`, creates separate tagged images for each architecture
- **Local vs. remote**: Without `--push`, images are saved to your local container engine (Docker, Podman, etc.)
- **Image size**: Consider using minimal base images (Alpine, distroless) for production deployments

---

## <span style="color: #E6622A;">version</span>

Display the current version of the gen-mcp CLI.

#### Usage

```bash
genmcp version
```

#### Output

```bash
# Release version
genmcp version v1.2.3

# Development version (when built from source)
genmcp version development@a1b2c3d
genmcp version development@a1b2c3d+uncommitedChanges
```

#### Examples

```bash
# Check version
genmcp version

# Use in scripts
VERSION=$(genmcp version | awk '{print $3}')
echo "Running gen-mcp $VERSION"

# Verify installation
which genmcp && genmcp version
```

---

## Common Workflows

#### Local Development

```bash
# 1. Convert an API (generates both files)
genmcp convert http://localhost:8080/openapi.json -o dev-tools.yaml
# Creates dev-tools.yaml and dev-server.yaml

# 2. Run and test
genmcp run -t dev-tools.yaml -s dev-server.yaml

# 3. Make changes to files, restart
# Press Ctrl+C, then run again
genmcp run -t dev-tools.yaml -s dev-server.yaml
```

#### Production Deployment

```bash
# 1. Validate configuration
genmcp run -t production-tools.yaml -s production-server.yaml
# Press Ctrl+C after confirming it starts

# 2. Build container
genmcp build -f production-tools.yaml -s production-server.yaml --tag myregistry/api:v1.0 --push

# 3. Deploy
kubectl apply -f k8s-deployment.yaml
```

#### Background Server Management

```bash
# Start server in background
genmcp run -t myapi.yaml -s myapi-server.yaml --detach

# Check if it's running (example using curl)
curl http://localhost:8080/health

# Stop when done (use tool definitions file)
genmcp stop -f myapi.yaml
```

#### Testing Multiple Configurations

```bash
# Test HTTP-based integration
genmcp run -t configs/http-tools.yaml -s configs/http-server.yaml -d

# Test CLI-based integration
genmcp run -t configs/cli-tools.yaml -s configs/cli-server.yaml -d

# Stop all (use tool definitions file paths)
genmcp stop -f configs/http-tools.yaml
genmcp stop -f configs/cli-tools.yaml
```

---

## Environment Variables

gen-mcp respects standard environment variables:

- **`HTTP_PROXY`, `HTTPS_PROXY`** - Used when fetching remote OpenAPI specs
- **`NO_PROXY`** - Bypass proxy for specified hosts
- **Container registry credentials** - Handled by your container runtime (Docker, Podman)

---

## Troubleshooting

#### Command Not Found

```bash
# Ensure genmcp is in PATH
echo $PATH
which genmcp

# If installed locally, use full path or add to PATH
./genmcp version
export PATH=$PATH:/path/to/genmcp
```

#### Server Won't Start

```bash
# Check MCP files validity
genmcp run -t mcpfile.yaml -s mcpserver.yaml
# Look for validation errors in output

# Verify files exist
ls -la mcpfile.yaml mcpserver.yaml

# Check port availability (for streamablehttp)
lsof -i :8080
```

#### Can't Stop Server

```bash
# Try with explicit tool definitions file path
genmcp stop -f /absolute/path/to/mcpfile.yaml

# Manually find and kill process
ps aux | grep genmcp
kill <pid>
```

#### Build Failures

```bash
# Ensure container runtime is running
docker info
# or
podman info

# Check registry authentication
docker login myregistry.com

# Try single platform first
genmcp build -f mcpfile.yaml -s mcpserver.yaml --tag test:latest --platform linux/amd64
```

---

## Getting Help

```bash
# General help
genmcp --help

# Command-specific help
genmcp run --help
genmcp convert --help
genmcp build --help
```

For more assistance:
- [Join our Discord](https://discord.gg/AwP6GAUEQR)
- [Report issues on GitHub](https://github.com/genmcp/gen-mcp/issues)
- [Read the documentation]({{ '/' | relative_url }})
