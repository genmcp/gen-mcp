# GenMCP Migration Guide

This guide helps you migrate between different versions of GenMCP configuration formats. Each section covers a specific version migration with step-by-step instructions, examples, and troubleshooting tips.

## Table of Contents

- [Migrating from 0.1.0 to 0.2.0](#migrating-from-010-to-020) - Single file to two-file format

---

## Migrating from 0.1.0 to 0.2.0

### Overview

GenMCP migrated from a single configuration file (`mcpfile.yaml`) to a two-file format:
- **`mcpfile.yaml`** - Defines capabilities (tools, prompts, resources, resource templates) and invocation bases
- **`mcpserver.yaml`** - Defines server runtime configuration (transport protocol, logging, authentication, TLS)

This separation allows you to:
- Share tool definitions across different server configurations
- Version tool definitions and server configuration independently
- Deploy the same tools with different runtime settings (dev, staging, production)

### What Changed

#### Schema Version
- **Old**: `mcpFileVersion: "0.1.0"`
- **New**: `schemaVersion: "0.2.0"` (in both files)

#### New Required Fields
- **`mcpfile.yaml`**: Must include `kind: "MCPToolDefinitions"`
- **`mcpserver.yaml`**: Must include `kind: "MCPServerConfig"`

#### Field Migration

| Old Location (mcpfile.yaml) | New Location                      |
|-----------------------------|-----------------------------------|
| `mcpFileVersion`            | `schemaVersion` (in mcpfile.yaml) |
| `name`                      | Stays in `mcpfile.yaml`           |
| `version`                   | Stays in `mcpfile.yaml`           |
| `instructions`              | Stays in `mcpfile.yaml`           |
| `runtime`                   | **Moved to `mcpserver.yaml`**     |
| `invocationBases`           | Stays in `mcpfile.yaml`           |
| `tools`                     | Stays in `mcpfile.yaml`           |
| `prompts`                   | Stays in `mcpfile.yaml`           |
| `resources`                 | Stays in `mcpfile.yaml`           |
| `resourceTemplates`         | Stays in `mcpfile.yaml`           |

### Step-by-Step Migration

#### Step 1: Create the New MCP File (`mcpfile.yaml`)

1. Copy your existing `mcpfile.yaml` to a temporary location as backup
2. Update the top-level fields:
   - Change `mcpFileVersion: "0.1.0"` to `schemaVersion: "0.2.0"`
   - Add `kind: "MCPToolDefinitions"` at the top
3. Remove the `runtime` section (entire section moves to `mcpserver.yaml`)
4. Keep all other fields

#### Step 2: Create the Server Config File (`mcpserver.yaml`)

1. Create a new file named `mcpserver.yaml`
2. Add the required top-level fields:
   - `kind: "MCPServerConfig"`
   - `schemaVersion: "0.2.0"`
3. Move the `runtime` section from your old `mcpfile.yaml` to this new file
4. If your old file had no `runtime` section, you can omit it (defaults to `streamablehttp` on port `3000`)

#### Step 3: Update Your Commands

When running `genmcp run`, you now need to specify both files:

```bash
# Old command (single file)
genmcp run --mcpfile mcpfile.yaml

# New command (two files)
genmcp run --mcpfile mcpfile.yaml --mcpserver mcpserver.yaml
```

### Migration Example

**Before** (`mcpfile.yaml` v0.1.0):
```yaml
mcpFileVersion: "0.1.0"
name: secure-server
version: "1.0.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8443
    tls:
      certFile: /etc/ssl/certs/server.crt
      keyFile: /etc/ssl/private/server.key
    auth:
      authorizationServers:
        - https://auth.example.com
      jwksUri: https://auth.example.com/.well-known/jwks.json
tools:
  - name: admin_tool
    description: "Administrative tool"
    requiredScopes:
      - admin:write
    inputSchema:
      type: object
    invocation:
      http:
        method: POST
        url: https://api.example.com/admin/action
```

**After** (`mcpfile.yaml` v0.2.0):
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: secure-server
version: "1.0.0"
tools:
  - name: admin_tool
    description: "Administrative tool"
    requiredScopes:
      - admin:write
    inputSchema:
      type: object
    invocation:
      http:
        method: POST
        url: https://api.example.com/admin/action
```

**After** (`mcpserver.yaml` v0.2.0):
```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8443
    tls:
      certFile: /etc/ssl/certs/server.crt
      keyFile: /etc/ssl/private/server.key
    auth:
      authorizationServers:
        - https://auth.example.com
      jwksUri: https://auth.example.com/.well-known/jwks.json
```
