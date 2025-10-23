# MCP Server Configuration Format Specification

## 1. Introduction

The MCP Server Configuration file (`mcpserver.yaml`) defines the runtime settings for an MCP server. This document details version `0.2.0` of the server configuration format.

## 2. MCPServerConfig

The server configuration file contains runtime settings for the MCP server.

| Field | Type | Description | Required |
|---|---|---|---|
| `kind` | string | Must be `"MCPServerConfig"`. | Yes |
| `schemaVersion` | string | The version of the schema format. Must be `"0.2.0"`. | Yes |
| `runtime` | `ServerRuntime` | The runtime settings for the server. | Yes |

### Example

```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
```

## 3. ServerRuntime Object

The `ServerRuntime` object specifies the transport protocol and its configuration for the server.

| Field | Type | Description | Required |
|---|---|---|---|
| `transportProtocol` | string | The transport protocol to use. Must be one of `streamablehttp` or `stdio`. | Yes |
| `streamableHttpConfig` | `StreamableHTTPConfig` | Configuration for the `streamablehttp` transport protocol. Required if `transportProtocol` is `streamablehttp`. | No |
| `stdioConfig` | `StdioConfig` | Configuration for the `stdio` transport protocol. Required if `transportProtocol` is `stdio`. | No |
| `loggingConfig` | `LoggingConfig` | Configuration for server logging. | No |

### 3.1. StreamableHTTPConfig Object

| Field | Type | Description | Required |
|---|---|---|---|
| `port` | integer | The port for the server to listen on. | Yes |
| `basePath` | string | The base path for the MCP server. Defaults to `/mcp`. | No |
| `stateless` | boolean | Indicates whether the server is stateless. | No |
| `auth` | `AuthConfig` | OAuth 2.0 configuration for protected resource. | No |
| `tls` | `TLSConfig` | TLS configuration for the HTTP server. | No |

### 3.2. TLSConfig Object

| Field | Type | Description | Required |
|---|---|---|---|
| `certFile` | string | The absolute path to the server's public certificate file on the runtime host where the MCP server will execute. | Yes |
| `keyFile` | string | The absolute path to the server's private key file on the runtime host where the MCP server will execute. | Yes |

### 3.3. AuthConfig Object

| Field | Type | Description | Required |
|---|---|---|---|
| `authorizationServers` | array of string | List of authorization server URLs for OAuth 2.0 token validation. | No |
| `jwksUri` | string | JSON Web Key Set URI for token signature verification. If no value is given but `authorizationServers` is set, gen-mcp will try to find a JWKS endpoint using different fallback paths.| No |

### 3.4. StdioConfig Object

This object is currently empty and serves as a placeholder for future configuration options.

### 3.5. LoggingConfig Object

| Field | Type | Description | Required |
|---|---|---|---|
| `level` | string | The minimum enabled logging level (debug, info, warn, error, dpanic, panic, fatal). | No |
| `development` | boolean | Puts the logger in development mode. | No |
| `disableCaller` | boolean | Stops annotating logs with the calling function's file name and line number. | No |
| `disableStacktrace` | boolean | Completely disables automatic stacktrace capturing. | No |
| `encoding` | string | Sets the logger's encoding ("json" or "console"). | No |
| `outputPaths` | array of string | A list of URLs or file paths to write logging output to. | No |
| `errorOutputPaths` | array of string | A list of URLs to write internal logger errors to. | No |
| `initialFields` | map[string]interface{} | A collection of fields to add to the root logger. | No |
| `enableMcpLogs` | boolean | Controls whether logs are sent to MCP clients. Defaults to true. | No |

**Note**: When `enableMcpLogs` is true, all MCP log entries are sent to MCP clients regardless of the configured `level`. The MCP client determines which log levels to actually display or process.

## 4. Complete Examples

### 4.1. Basic Server with Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8008
  loggingConfig:
    enableMcpLogs: true
    encoding: "console"
    level: "debug"
```

### 4.2. Production Server with Advanced Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
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

### 4.3. Server with Disabled MCP Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
  loggingConfig:
    level: "warn"
    encoding: "json"
    enableMcpLogs: false
    outputPaths:
      - "/var/log/system.log"
```

## 5. Security Configuration Examples

### 5.1. TLS Configuration

To enable HTTPS for your MCP server, configure TLS in the `streamableHttpConfig`:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8443
    tls:
      certFile: /etc/ssl/certs/server.crt
      keyFile: /etc/ssl/private/server.key
```

### 5.2. OAuth 2.0 Configuration

To protect your MCP server with OAuth 2.0 authentication:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
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

For maximum security, combine both TLS and OAuth:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
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

## 6. STDIO Transport Configuration

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  transportProtocol: stdio
```
