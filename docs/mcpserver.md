---
layout: page
title: GenMCP Config File Format - Server Config
description: Complete reference guide for the GenMCP Server Config File format
---

# GenMCP Server Config File Format

## 1. Introduction

GenMCP uses **two separate YAML configuration files** to define an MCP server:

1. **MCP File** (`mcpfile.yaml`) - Defines the capabilities (tools, prompts, resources, resource templates) and invocation bases
2. **Server Config File** (`mcpserver.yaml`) - Defines the server runtime configuration (transport protocol, logging, authentication, TLS)

This separation allows you to:
- Share tool definitions across different server configurations
- Version tool definitions and server configuration independently
- Deploy the same tools with different runtime settings (dev, staging, production)

The Server Config File uses schema version `0.2.0` and must be provided when running an MCP server with the `genmcp run` command.

> **Migrating from 0.1.0?** If you're upgrading from the single-file format (schema version 0.1.0), see the [Migration Guide](../MIGRATION.md) for step-by-step instructions.

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
| `clientTlsConfig`      | `ClientTLSConfig`      | TLS configuration for outbound HTTP requests (e.g., custom CA certificates).                                    | No       |

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

### 3.5. ClientTLSConfig Object

The `ClientTLSConfig` object configures TLS settings for **outbound** HTTP requests made by the MCP server (e.g., when tools invoke external APIs). This is useful when connecting to internal services that use certificates signed by a corporate or private Certificate Authority (CA).

| Field                | Type            | Description                                                                                                     | Required |
|----------------------|-----------------|-----------------------------------------------------------------------------------------------------------------|----------|
| `caCertFiles`        | array of string | Paths to CA certificate files (PEM format) to trust for outbound HTTPS requests.                               | No       |
| `caCertDir`          | string          | Path to a directory containing CA certificate files. All `.pem` and `.crt` files will be loaded.               | No       |
| `insecureSkipVerify` | boolean         | Skip TLS certificate verification. **WARNING: Insecure, use only for testing.**                                | No       |

**Note**: The CA certificates specified here are added to the system's default certificate pool, so standard public CAs remain trusted.

**Environment Variable Overrides**: These settings can also be configured via environment variables:
- `GENMCP_CLIENTTLSCONFIG_CACERTFILES=/path/to/ca1.pem,/path/to/ca2.pem`
- `GENMCP_CLIENTTLSCONFIG_CACERTDIR=/etc/ssl/certs/custom/`
- `GENMCP_CLIENTTLSCONFIG_INSECURESKIPVERIFY=true`

### 3.6. LoggingConfig Object

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

### 5.4. Custom CA Certificates for Outbound Requests

When your MCP server needs to make HTTPS requests to internal services that use certificates signed by a corporate or private CA, configure `clientTlsConfig`:

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
  clientTlsConfig:
    # Option 1: Specify individual CA certificate files
    caCertFiles:
      - /etc/pki/tls/certs/ca-bundle.crt
      - /etc/ssl/certs/corporate-ca.pem

    # Option 2: Specify a directory containing CA certificates
    caCertDir: /etc/ssl/certs/custom-cas/
```

**Use Case**: This is commonly needed when:
- Running in a Kubernetes cluster with a service mesh (e.g., Istio) that uses internal CAs
- Connecting to internal APIs behind a corporate proxy
- Accessing services with self-signed certificates in development/staging environments

**Kubernetes Example** - Mount custom CA certificates from a Secret or ConfigMap:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mcp-server
spec:
  containers:
    - name: mcp-server
      image: my-mcp-server:latest
      volumeMounts:
        - name: ca-certs
          mountPath: /etc/ssl/certs/custom-cas
          readOnly: true
  volumes:
    - name: ca-certs
      secret:
        secretName: corporate-ca-bundle
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

