---
layout: page
title: Command Reference
description: Complete reference for all gen-mcp CLI commands
---

# Command Reference

The `genmcp` CLI provides commands for managing MCP servers, converting API specifications, and building container images. This guide covers all available commands with detailed explanations and examples.

---

## Quick Reference

| Command               | Description             | Common Usage                                                        |
|-----------------------|-------------------------|---------------------------------------------------------------------|
| [`run`](#run)         | Start an MCP server     | `genmcp run -f mcpfile.yaml -s mcpserver.yaml`                      |
| [`stop`](#stop)       | Stop a running server   | `genmcp stop -f mcpfile.yaml`                                       |
| [`inspect`](#inspect) | Show server details     | `genmcp inspect -s mcpserver.yaml`                                  |
| [`convert`](#convert) | Convert OpenAPI to MCP  | `genmcp convert openapi.json`                                       |
| [`build`](#build)     | Build container image   | `genmcp build -f mcpfile.yaml -s mcpserver.yaml --tag myapi:latest` |
| [`version`](#version) | Display version info    | `genmcp version`                                                    |

---

## <span style="color: #E6622A;">run</span>

Start an MCP server from GenMCP config files.

#### Usage

```bash
genmcp run [flags]
```

#### Flags

| Flag              | Short | Default          | Description                                      |
|-------------------|-------|------------------|--------------------------------------------------|
| `--file`          | `-f`  | `mcpfile.yaml`   | Path to the MCP File (MCPToolDefinitions)        |
| `--server-config` | `-s`  | `mcpserver.yaml` | Path to the server config file (MCPServerConfig) |
| `--detach`        | `-d`  | `false`          | Run server in background (detached mode)         |

#### How It Works

The `run` command:

1. **Validates both files** - Checks syntax and schema validity of both the MCP file and the server config file
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
genmcp run -f ./config/tools.yaml -s ./config/server.yaml

# Run with absolute paths
genmcp run -f /path/to/mcpfile.yaml -s /path/to/mcpserver.yaml
```

**Detached mode (background):**
```bash
# Start server in background
genmcp run -f mcpfile.yaml -s mcpserver.yaml --detach

# Server runs independently, can close terminal
# Use 'genmcp stop' to terminate later
```

**Real-world scenarios:**

```bash
# Development: Run in foreground with logs visible
cd examples/ollama
genmcp run -f ollama-http-mcpfile.yaml -s ollama-mcpserver.yaml

# Production: Run in background
genmcp run -f /etc/genmcp/tools.yaml -s /etc/genmcp/server.yaml -d

# Testing: Quick validation and startup
genmcp run -f test-tools.yaml -s test-server.yaml
# Press Ctrl+C to stop when done testing
```

#### Notes

- **Two files required**: Both the MCP file and the server config file must be provided
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

| Flag     | Short | Default        | Description                                                     |
|----------|-------|----------------|-----------------------------------------------------------------|
| `--file` | `-f`  | `mcpfile.yaml` | Path to the MCP file of the server to stop (used as identifier) |

#### How It Works

The `stop` command:

1. **Resolves the MCP file path** - Finds the absolute path to match the running server (uses MCP file as identifier)
2. **Retrieves the process ID** - Looks up the saved PID from when the server was started
3. **Terminates the process** - Sends a kill signal to stop the server
4. **Cleans up** - Removes the saved process ID

#### Examples

**Basic usage:**
```bash
# Stop server using default MCP file (mcpfile.yaml)
genmcp stop

# Stop server with specific MCP file
genmcp stop -f mcpfile.yaml

# Stop server with absolute path
genmcp stop -f /path/to/mcpfile.yaml
```

**Workflow example:**
```bash
# Start server in background
genmcp run -f myapi.yaml -s myapi-server.yaml --detach
# Output: successfully started gen-mcp server...

# Later, stop the server (use MCP file path)
genmcp stop -f myapi.yaml
# Output: successfully stopped gen-mcp server...
```

#### Notes

- **File path must match**: The MCP file path used with `stop` must match the path used with `run --detach`
- **Only works with detached servers**: Servers running in foreground mode can be stopped with Ctrl+C
- **Manual cleanup**: If the process was manually killed outside gen-mcp, you may need to manually clean up the saved PID file

---

## <span style="color: #E6622A;">inspect</span>

Display detailed information about an MCP server configuration.

#### Usage

```bash
genmcp inspect [name] [flags]
```

#### Arguments

| Argument | Description                                           |
|----------|-------------------------------------------------------|
| `[name]` | Optional name of a running server to inspect          |

#### Flags

| Flag              | Short | Default          | Description                                      |
|-------------------|-------|------------------|--------------------------------------------------|
| `--file`          | `-f`  | `mcpfile.yaml`   | Path to the MCP file                             |
| `--server-config` | `-s`  | `mcpserver.yaml` | Path to the server config file                   |
| `--json`          |       | `false`          | Output in JSON format for machine-readable output|

#### How It Works

The `inspect` command:

1. **Loads configuration** - Parses both the MCP file and server config file
2. **Displays server info** - Shows name, version, and transport configuration
3. **Lists capabilities** - Shows all tools, prompts, resources, and resource templates with descriptions
4. **Shows security status** - Indicates whether TLS and OAuth authentication are configured
5. **Generates client config** - Outputs valid JSON for configuring MCP clients to connect

**Lookup by Name:**
When a server name is provided as an argument, the command looks up running servers by name from the process registry (servers started with `genmcp run --detach`).

**Smart Path Resolution:**
When only `-s` is specified without `-f`, the command automatically looks for `mcpfile.yaml` in the same directory as the server config file.

#### Examples

**Basic usage:**
```bash
# Inspect using default files (mcpfile.yaml and mcpserver.yaml)
genmcp inspect

# Inspect with specific server config (auto-finds mcpfile.yaml in same directory)
genmcp inspect -s examples/http-conversion/mcpserver.yaml

# Inspect with explicit file paths
genmcp inspect -f myapi.yaml -s myapi-server.yaml
```

**JSON output for scripting:**
```bash
# Get machine-readable output
genmcp inspect -s mcpserver.yaml --json

# Extract specific fields with jq
genmcp inspect -s mcpserver.yaml --json | jq '.tools[].name'

# Get MCP client configuration
genmcp inspect -s mcpserver.yaml --json | jq '.mcpClientConfig'
```

**Inspect a running server:**
```bash
# Start a server in background
genmcp run -f myapi.yaml -s myapi-server.yaml --detach

# Inspect by server name
genmcp inspect "My API Server"

# If server not found, shows available running servers
genmcp inspect "Unknown Server"
# Output: no running server found with name: Unknown Server
#         Available running servers:
#           - My API Server (PID: 12345)
```

#### Output Format

**Human-readable output:**
```
Server: Feature Request API (v1.0.0)
Transport: streamablehttp
Endpoint: http://localhost:8080/mcp

Security:
  TLS: disabled
  Auth: enabled (OAuth 2.0)

Health Endpoints:
  Liveness: /healthz
  Readiness: /readyz

Capabilities:
  Tools (5):
    - get_features: Returns a list of all features
    - post_features: Create a new feature request
    ...
  Prompts (1):
    - feature-analysis: Analyze feature requests
  Resources (1):
    - feature-report: Feature progress report (uri: http://...)

MCP Client Configuration:
  {
    "mcpServers": {
      "Feature Request API": {
        "type": "http",
        "url": "http://localhost:8080/mcp"
      }
    }
  }
```

**JSON output (`--json`):**
```json
{
  "server": {
    "name": "Feature Request API",
    "version": "1.0.0"
  },
  "transport": {
    "protocol": "streamablehttp",
    "port": 8080,
    "basePath": "/mcp"
  },
  "security": {
    "auth": {
      "enabled": true,
      "jwksUri": "https://auth.example.com/.well-known/jwks.json"
    }
  },
  "tools": [...],
  "prompts": [...],
  "resources": [...],
  "mcpClientConfig": {
    "mcpServers": {
      "Feature Request API": {
        "type": "http",
        "url": "http://localhost:8080/mcp"
      }
    }
  }
}
```

#### Notes

- **MCP Client Configuration**: The generated JSON can be used directly with Claude Desktop, Claude Code, or other MCP clients
- **Security display**: Shows whether TLS and Auth are enabled without exposing sensitive values like keys or secrets
- **Transport-aware**: Generates appropriate client config for both HTTP and stdio transports

---

## <span style="color: #E6622A;">convert</span>

Convert an OpenAPI v2 or v3 specification into GenMCP config files.

#### Usage

```bash
genmcp convert <openapi-spec> [flags]
```

#### Arguments

| Argument         | Description                                              |
|------------------|----------------------------------------------------------|
| `<openapi-spec>` | URL or file path to OpenAPI specification (JSON or YAML) |

#### Flags

| Flag              | Short | Default          | Description                                      |
|-------------------|-------|------------------|--------------------------------------------------|
| `--file`          | `-f`  | `mcpfile.yaml`   | Output path for the generated MCP file           |
| `--server-config` | `-s`  | `mcpserver.yaml` | Output path for the generated server config file |
| `--host`          | `-H`  | *(from spec)*    | Override the base host URL from the OpenAPI spec |

#### How It Works

The `convert` command:

1. **Fetches the spec** - Downloads from URL or reads from file
2. **Parses OpenAPI** - Supports both OpenAPI v2 (Swagger) and v3 formats
3. **Generates tools** - Creates an MCP tool for each API endpoint
4. **Maps schemas** - Converts OpenAPI parameter schemas to JSON Schema for input validation
5. **Creates invocations** - Generates HTTP invocations with proper methods and URLs
6. **Writes GenMCP config files** - Outputs both an MCP file and a server config file

**File Naming Convention:**
- The `--file/-f` flag sets the output path for the MCP file (default: `mcpfile.yaml`)
- The `--server-config/-s` flag sets the output path for the server config file (default: `mcpserver.yaml`)
- If you only specify `--file/-f`, the server config file will use the default name `mcpserver.yaml` regardless of the MCP filename
- To control both filenames, specify both `--file/-f` and `--server-config/-s` flags

#### Examples

**Convert from URL:**
```bash
# Public API (uses default filenames: mcpfile.yaml and mcpserver.yaml)
genmcp convert https://petstore.swagger.io/v2/swagger.json

# Local server
genmcp convert http://localhost:8080/openapi.json

# With custom tool definitions output path (server config defaults to mcpserver.yaml)
genmcp convert https://api.example.com/openapi.yaml -f my-api.yaml
# Creates: my-api.yaml and mcpserver.yaml

# With custom output paths for both files
genmcp convert https://api.example.com/openapi.yaml -f my-api.yaml -s my-api-server.yaml
# Creates: my-api.yaml and my-api-server.yaml
```

**Convert from file:**
```bash
# Local OpenAPI file (uses default filenames)
genmcp convert ./api-spec.json

# With custom tool definitions location (server config defaults to mcpserver.yaml)
genmcp convert ./specs/v3-api.yaml -f ./configs/mcp-api.yaml
# Creates: ./configs/mcp-api.yaml and mcpserver.yaml

# With custom paths for both files
genmcp convert ./specs/v3-api.yaml -f ./configs/mcp-api.yaml -s ./configs/mcp-api-server.yaml
# Creates: ./configs/mcp-api.yaml and ./configs/mcp-api-server.yaml
```

**Override host URL:**
```bash
# Original spec has https://api.example.com
# Override to use local dev server
genmcp convert openapi.json --host http://localhost:3000

# Override to use staging environment with custom output paths
genmcp convert openapi.json -H https://staging-api.example.com -f staging.yaml -s staging-server.yaml
```

**Complete workflow:**
```bash
# 1. Convert OpenAPI spec (generates both files)
# Using --file/-f sets tool definitions path; server config defaults to mcpserver.yaml
genmcp convert https://api.github.com/openapi.json -f github-tools.yaml
# Output: wrote tool definitions to github-tools.yaml
# Output: wrote server config to mcpserver.yaml

# Or specify both files explicitly
genmcp convert https://api.github.com/openapi.json -f github-tools.yaml -s github-server.yaml
# Output: wrote tool definitions to github-tools.yaml
# Output: wrote server config to github-server.yaml

# 2. Review and customize the generated files
cat github-tools.yaml
cat github-server.yaml
# Edit descriptions, add safety guards, etc.

# 3. Run the MCP server
genmcp run -f github-tools.yaml -s github-server.yaml
```

#### Generated Structure

The converter automatically creates two files:

**MCP File** (`mcpfile.yaml`):

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

| Flag              | Short | Default          | Description                                       |
|-------------------|-------|------------------|---------------------------------------------------|
| `--file`          | `-f`  | `mcpfile.yaml`   | Path to MCP file to include in image |
| `--server-config` | `-s`  | `mcpserver.yaml` | Path to server config file to include in image    |
| `--tag`           |       | *(required)*     | Image tag (e.g., `myregistry/myapi:v1.0`)         |
| `--base-image`    |       | *(auto)*         | Base container image to build on                  |
| `--platform`      |       | `multi-arch`     | Target platform (e.g., `linux/amd64`)             |
| `--push`          |       | `false`          | Push to registry instead of saving locally        |
| `--server-version`|       | *(auto)*         | Server binary version to download (default: latest for dev builds, CLI version for releases) |

#### How It Works

The `build` command:

1. **Downloads server binaries** - Fetches required binaries from GitHub releases (with cosign verification)
2. **Validates both GenMCP config files** - Ensures both tool definitions and server config are valid
3. **Builds container image** - Creates a containerized MCP server with both files included
4. **Supports multi-arch** - By default builds for `linux/amd64` and `linux/arm64`
5. **Saves or pushes** - Either stores locally or pushes to a container registry

**Binary Management:**
- Server binaries are downloaded from GitHub releases and cached locally
- Downloaded binaries are cryptographically verified with Sigstore for security (built-in)
- Cache location: 
  - Linux: `~/.cache/genmcp/binaries/`
  - macOS: `~/Library/Caches/.genmcp/binaries/`
  - Windows: `%LOCALAPPDATA%\.genmcp\binaries\`
- **Version Matching**: 
  - Release CLI versions download matching server binaries
  - Development CLI versions automatically use latest release
  - Use `--server-version` to override
- **Requirements**: Network access to GitHub releases and API

#### Examples

**Basic local build:**
```bash
# Build and save to local Docker daemon (uses default files)
genmcp build --tag myapi:latest

# Build with specific files
genmcp build -f config/tools.yaml -s config/server.yaml --tag myapi:v1.0

# Build for specific platform only
genmcp build --tag myapi:latest --platform linux/amd64

# Build with specific server version
genmcp build --tag myapi:latest --server-version v0.1.0
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
# Using --file/-f sets tool definitions path; server config defaults to mcpserver.yaml
genmcp convert http://localhost:8080/openapi.json -f dev-tools.yaml
# Creates dev-tools.yaml and mcpserver.yaml

# Or specify both files explicitly
genmcp convert http://localhost:8080/openapi.json -f dev-tools.yaml -s dev-server.yaml
# Creates dev-tools.yaml and dev-server.yaml

# 2. Run and test
genmcp run -f dev-tools.yaml -s dev-server.yaml

# 3. Make changes to files, restart
# Press Ctrl+C, then run again
genmcp run -f dev-tools.yaml -s dev-server.yaml
```

#### Production Deployment

```bash
# 1. Validate configuration
genmcp run -f production-tools.yaml -s production-server.yaml
# Press Ctrl+C after confirming it starts

# 2. Build container
genmcp build -f production-tools.yaml -s production-server.yaml --tag myregistry/api:v1.0 --push

# 3. Deploy
kubectl apply -f k8s-deployment.yaml
```

#### Background Server Management

```bash
# Start server in background
genmcp run -f myapi.yaml -s myapi-server.yaml --detach

# Check if it's running (example using curl)
curl http://localhost:8080/health

# Stop when done (use MCP file)
genmcp stop -f myapi.yaml
```

#### Testing Multiple Configurations

```bash
# Test HTTP-based integration
genmcp run -f configs/http-tools.yaml -s configs/http-server.yaml -d

# Test CLI-based integration
genmcp run -f configs/cli-tools.yaml -s configs/cli-server.yaml -d

# Stop all (use MCP file paths)
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
# Check GenMCP config files validity
genmcp run -f mcpfile.yaml -s mcpserver.yaml
# Look for validation errors in output

# Verify files exist
ls -la mcpfile.yaml mcpserver.yaml

# Check port availability (for streamablehttp)
lsof -i :8080
```

#### Can't Stop Server

```bash
# Try with explicit MCP file path
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
