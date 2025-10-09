# MCP Server Configuration Specification

## 1. Introduction

The MCP Server Configuration file (typically `mcpserver.yaml`) is a YAML-based configuration that defines the runtime settings for an MCP server. This file is separate from the `mcpfile.yaml` which contains tool, prompt, and resource definitions. This separation allows for:

- Clear separation of concerns between runtime settings and capability definitions
- Easier management of server deployment configurations
- Future extensibility for logging, tracing, and other operational settings

This document details version `0.1.0` of the server configuration format.

## 2. File Structure

The server configuration file has the following top-level fields:

| Field | Type | Description | Required |
|---|---|---|---|
| `mcpFileVersion` | string | The version of the MCP file format. Must be `"0.1.0"`. | Yes |
| `name` | string | The name of the server. | Yes |
| `version` | string | The semantic version of the server. | Yes |
| `runtime` | `ServerRuntime` | The runtime settings for the server. | Yes |

### Example

```yaml
mcpFileVersion: 0.1.0
name: my-awesome-server
version: 1.2.3
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
    basePath: /mcp
```

## 3. ServerRuntime Object

The `ServerRuntime` object specifies the transport protocol and its configuration for the server.

| Field | Type | Description | Required |
|---|---|---|---|
| `transportProtocol` | string | The transport protocol to use. Must be one of `streamablehttp` or `stdio`. | Yes |
| `streamableHttpConfig` | `StreamableHTTPConfig` | Configuration for the `streamablehttp` transport protocol. Required if `transportProtocol` is `streamablehttp`. | No |
| `stdioConfig` | `StdioConfig` | Configuration for the `stdio` transport protocol. Required if `transportProtocol` is `stdio`. | No |

### 3.1. StreamableHTTPConfig Object

| Field | Type | Description | Required |
|---|---|---|---|
| `port` | integer | The port for the server to listen on. | Yes |
| `basePath` | string | The base path for the MCP server. Defaults to `/mcp`. | No |
| `stateless` | boolean | Whether the server operates in stateless mode. Defaults to `true`. | No |
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
| `jwksUri` | string | JSON Web Key Set URI for token signature verification. If no value is given but `authorizationServers` is set, gen-mcp will try to find a JWKS endpoint using different fallback paths. | No |

### 3.4. StdioConfig Object

This object is currently empty and serves as a placeholder for future configuration options.

## 4. Usage

To run an MCP server with separate configuration files:

```bash
gen-mcp run -f mcpfile.yaml -s mcpserver.yaml
```

Where:
- `mcpfile.yaml` contains tool/prompt/resource definitions
- `mcpserver.yaml` contains runtime configuration (REQUIRED)

## 5. Complete Example

### mcpserver.yaml
```yaml
mcpFileVersion: "0.1.0"
name: user-service
version: "2.1.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
    basePath: /mcp
    stateless: true
    tls:
      certFile: /etc/ssl/certs/server.crt
      keyFile: /etc/ssl/private/server.key
```

### mcpfile.yaml
```yaml
mcpFileVersion: "0.1.0"
tools:
- name: get_user
  title: "Get User"
  description: "Retrieves a user by their ID."
  inputSchema:
    type: object
    properties:
      userId:
        type: string
        description: "The ID of the user to retrieve."
    required:
    - userId
  invocation:
    http:
      method: GET
      url: http://localhost:8080/users/{userId}
```

## 6. Future Enhancements

The server configuration format is designed to be extensible for future operational settings such as:

- Logging configuration (levels, formats, outputs)
- Tracing and observability settings
- Performance tuning parameters
- Resource limits
