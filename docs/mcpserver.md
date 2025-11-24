---
layout: page
title: GenMCP Config File Format - Server Config
description: Complete reference guide for the GenMCP Server Config File format
---

# GenMCP Server Config File Format

## 1. Introduction

GenMCP uses **two separate YAML configuration files** to define an MCP server:

1. **Tool Definitions File** (`mcpfile.yaml`) - Defines the capabilities (tools, prompts, resources, resource templates) and invocation bases
2. **Server Config File** (`mcpserver.yaml`) - Defines the server runtime configuration (transport protocol, logging, authentication, TLS)

This separation allows you to:
- Share tool definitions across different server configurations
- Version tool definitions and server configuration independently
- Deploy the same tools with different runtime settings (dev, staging, production)

The Server Config File uses schema version `0.2.0` and must be provided when running an MCP server with the `genmcp run` command.

## 2. Server Config File

The Server Config File defines how the MCP server runs, including transport protocol, logging, authentication, and TLS settings.

### 2.1. Top-Level Object

The root of the Server Config File has the following structure:

| Field             | Type            | Description                                                                                                 | Required |
|-------------------|-----------------|-------------------------------------------------------------------------------------------------------------|----------|
| `kind`            | string          | Must be `"MCPServerConfig"`.                                                                                | Yes      |
| `schemaVersion`   | string          | The version of the GenMCP config file format. Must be `"0.2.0"`.                                                      | Yes      |
| `runtime`         | `ServerRuntime` | The runtime settings for the server. If omitted, defaults to `streamablehttp` on port `3000`.               | No       |

### Example: Server Config File

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
    basePath: /mcp
  loggingConfig:
    level: info
    encoding: json
    enableMcpLogs: true
```

## 3. ServerRuntime Object

The `ServerRuntime` object specifies the transport protocol and its configuration for the server.

| Field                  | Type                   | Description                                                                                                     | Required |
|------------------------|------------------------|-----------------------------------------------------------------------------------------------------------------|----------|
| `transportProtocol`    | string                 | The transport protocol to use. Must be one of `streamablehttp` or `stdio`.                                      | Yes      |
| `streamableHttpConfig` | `StreamableHTTPConfig` | Configuration for the `streamablehttp` transport protocol. Required if `transportProtocol` is `streamablehttp`. | No       |
| `stdioConfig`          | `StdioConfig`          | Configuration for the `stdio` transport protocol. Required if `transportProtocol` is `stdio`.                   | No       |
| `loggingConfig`        | `LoggingConfig`        | Configuration for server logging.                                                                               | No       |

### 3.1. StreamableHTTPConfig Object

| Field       | Type         | Description                                                    | Required |
|-------------|--------------|----------------------------------------------------------------|----------|
| `port`      | integer      | The port for the server to listen on.                          | Yes      |
| `basePath`  | string       | The base path for the MCP server. Defaults to `/mcp`.          | No       |
| `stateless` | boolean      | Indicates whether the server is stateless. Defaults to `true`. | No       |
| `auth`      | `AuthConfig` | OAuth 2.0 configuration for protected resources.               | No       |
| `tls`       | `TLSConfig`  | TLS configuration for HTTPS.                                   | No       |

### 3.2. TLSConfig Object

| Field      | Type   | Description                                                                                                      | Required |
|------------|--------|------------------------------------------------------------------------------------------------------------------|----------|
| `certFile` | string | The absolute path to the server's public certificate file on the runtime host where the MCP server will execute. | Yes      |
| `keyFile`  | string | The absolute path to the server's private key file on the runtime host where the MCP server will execute.        | Yes      |

### 3.3. AuthConfig Object

| Field                  | Type            | Description                                                                                                                                                                             | Required |
|------------------------|-----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| `authorizationServers` | array of string | List of authorization server URLs for OAuth 2.0 token validation.                                                                                                                       | No       |
| `jwksUri`              | string          | JSON Web Key Set URI for token signature verification. If no value is given but `authorizationServers` is set, gen-mcp will try to find a JWKS endpoint using different fallback paths. | No       |

### 3.4. StdioConfig Object

This object is currently empty and serves as a placeholder for future configuration options.

### 3.5. LoggingConfig Object

| Field               | Type                   | Description                                                                         | Required |
|---------------------|------------------------|-------------------------------------------------------------------------------------|----------|
| `level`             | string                 | The minimum enabled logging level (debug, info, warn, error, dpanic, panic, fatal). | No       |
| `development`       | boolean                | Puts the logger in development mode.                                                | No       |
| `disableCaller`     | boolean                | Stops annotating logs with the calling function's file name and line number.        | No       |
| `disableStacktrace` | boolean                | Completely disables automatic stacktrace capturing.                                 | No       |
| `encoding`          | string                 | Sets the logger's encoding ("json" or "console").                                   | No       |
| `outputPaths`       | array of string        | A list of URLs or file paths to write logging output to.                            | No       |
| `errorOutputPaths`  | array of string        | A list of URLs to write internal logger errors to.                                  | No       |
| `initialFields`     | map[string]interface{} | A collection of fields to add to the root logger.                                   | No       |
| `enableMcpLogs`     | boolean                | Controls whether logs are sent to MCP clients. Defaults to true.                    | No       |

**Note**: When `enableMcpLogs` is true, all MCP log entries are sent to MCP clients regardless of the configured `level`. The MCP client determines which log levels to actually display or process.

## 4. Complete Examples

### 4.1. Basic Example

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8008
  loggingConfig:
    enableMcpLogs: true
    encoding: "console"
    level: "debug"
```

### 4.2. Advanced Logging Configuration

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
  loggingConfig:
    level: "info"
    encoding: "json"
    development: false
    disableCaller: false
    disableStacktrace: false
    outputPaths:
      - "/var/log/mcp-server.log"
      - "stdout"
    errorOutputPaths:
      - "/var/log/mcp-server-error.log"
      - "stderr"
    initialFields:
      service: "mcp-api"
      version: "1.0.0"
    enableMcpLogs: true
```

### 4.3. Using Invocation Bases and Extends

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
```

## 5. Security Configuration Examples

### 5.1. TLS Configuration

**Server Config File** (`mcpserver.yaml`):

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
```

### 5.2. OAuth 2.0 Configuration

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
    auth:
      authorizationServers:
        - https://auth.example.com
        - https://keycloak.company.com/auth/realms/mcp
      jwksUri: https://auth.example.com/.well-known/jwks.json
```

### 5.3. Combined TLS and OAuth Configuration

**Server Config File** (`mcpserver.yaml`):

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

## 6. Complete Example for a CLI

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: stdio
```

## 7. Complete Example for an HTTP Server

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 3000
```

