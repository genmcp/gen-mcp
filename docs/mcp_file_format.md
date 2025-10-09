# MCP File Format Specification

## 1. Introduction

The MCP (Model Context Protocol) file is a YAML-based configuration that defines the capabilities of an MCP server. It specifies the tools, prompts, and resources available, their input and output schemas, and how they should be invoked. This document details version `0.1.0` of the file format.

**Note**: As of version 0.1.0, runtime configuration (such as transport protocol, port, etc.) can be separated into a dedicated `mcpserver.yaml` file. See [Server Configuration Specification](./server_config_format.md) for details. For backward compatibility, runtime configuration can still be included in the `mcpfile.yaml`.

## 2. Top-Level Object

The root of the configuration is a single top-level object with the following fields:

| Field | Type | Description | Required |
|---|---|---|---|
| `mcpFileVersion` | string | The version of the MCP file format. Must be `"0.1.0"`. | Yes |
| `name` | string | The name of the server. Required if runtime is included in this file. | No* |
| `version` | string | The semantic version of the server's toolset. Required if runtime is included in this file. | No* |
| `runtime` | `ServerRuntime` | The runtime settings for the server. If omitted, runtime must be provided in a separate server config file. For backward compatibility, if included, it defaults to `streamablehttp` on port `3000`. | No |
| `tools` | array of `Tool` | The tools provided by this server. | No |
| `prompts` | array of `Prompt` | The prompts provided by this server. | No |
| `resources` | array of `Resource` | The resources provided by this server. | No |
| `resourceTemplates` | array of `ResourceTemplate` | The resource templates provided by this server. | No |

*Note: `name` and `version` are required when `runtime` is present in this file, or when using this file standalone without a separate server config.

### Example (Traditional - with runtime)

```yaml
mcpFileVersion: 0.1.0
name: my-awesome-server
version: 1.2.3
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
tools:
  # ... tool definitions
```

### Example (New - without runtime)

When using a separate server configuration file:

**mcpfile.yaml:**
```yaml
mcpFileVersion: 0.1.0
tools:
- name: get_user
  description: "Retrieves a user by their ID"
  inputSchema:
    type: object
    properties:
      userId:
        type: string
    required:
    - userId
  invocation:
    http:
      method: GET
      url: http://localhost:8080/users/{userId}
```

**mcpserver.yaml:**
```yaml
mcpFileVersion: 0.1.0
name: my-awesome-server
version: 1.2.3
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
```

Run with: `gen-mcp run -f mcpfile.yaml -s mcpserver.yaml`

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

## 4. Tool Object

A `Tool` object describes a specific, invokable function.

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | A unique, programmatic identifier for the tool (e.g., `clone_repo`). | Yes |
| `title` | string | A human-readable title for display purposes (e.g., "Clone Git Repository"). | No |
| `description` | string | A detailed description of what the tool does, intended for an LLM to understand its function. | Yes |
| `inputSchema` | `JsonSchema` | A JSON Schema object defining the parameters the tool accepts. | Yes |
| `outputSchema` | `JsonSchema` | A JSON Schema object defining the structure of the tool's output. | No |
| `invocation` | `Invocation` | An object describing how to execute the tool. Must contain a single key: either `http` or `cli`. | Yes |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to execute this tool. Only relevant when the server uses OAuth authentication. | No |

## 5. JsonSchema Object

The `inputSchema` and `outputSchema` fields use the JSON Schema standard to define data structures.

| Field | Type | Description |
|---|---|---|
| `type` | string | The data type. Can be `string`, `number`, `integer`, `boolean`, `array`, `object`, or `null`. |
| `description` | string | A human-readable description of the schema or field. |
| `properties` | map[string]`JsonSchema` | For `object` types, defines the named properties of the object. |
| `required` | array of string | For `object` types, lists the property names that are required. |
| `items` | `JsonSchema` | For `array` types, defines the schema of each item in the array. |
| `additionalProperties`| boolean | For `object` types, specifies whether additional properties are allowed. |

### Example

```yaml
inputSchema:
  type: object
  properties:
    location:
      type: string
      description: "The city and state, e.g., San Francisco, CA"
  required:
    - location
```

## 6. Invocation Object

The `invocation` object specifies how a tool is executed. It must contain exactly one of the following keys.

### 6.1. HTTP Invocation

The `http` invocation type is used for tools that are called via an HTTP request.

| Field | Type | Description | Required |
|---|---|---|---|
| `method` | string | The HTTP method (e.g., `GET`, `POST`). | Yes |
| `url` | string | The URL to send the request to. It can be a template. Input parameters from the `inputSchema` are substituted into placeholders like `{paramName}`. | Yes |

#### Example

```yaml
invocation:
  http:
    method: GET
    url: http://localhost:8080/users/{userId}
```

### 6.2. CLI Invocation

The `cli` invocation type is used for tools that are executed via a shell command.

| Field | Type | Description | Required |
|---|---|---|---|
| `command` | string | The command to execute. It can be a template with placeholders like `{placeholder}` that correspond to keys in the `templateVariables` map. | Yes |
| `templateVariables` | map[string]`TemplateVariable` | A map defining how `inputSchema` properties are formatted into command-line arguments. If a placeholder is present in `command` but not in `templateVariables`, the value of the property of the same name in the `inputSchema` will be used. | No |

#### TemplateVariable Object

The `templateVariables` map links placeholders in the `command` string to properties in the `inputSchema`. The **key** of the map must match a placeholder in the `command` string.

| Field | Type | Description | Required |
|---|---|---|---|
| `property` | string | The name of the property in the `inputSchema` to use for this variable. | Yes |
| `format` | string | A format string for the argument (e.g., `--depth {depth}`). The placeholder within this string is replaced by the actual value of the input property. | No |
| `omitIfFalse` | boolean | If `true`, the entire formatted argument is omitted from the command if the input property's value is `false`. Defaults to `false`. | No |

#### Example

In this example, the `{repoUrl}`, `{depth}`, and `{verbose}` placeholders in `command` are defined in `templateVariables`. Each `TemplateVariable` then maps to a property in the `inputSchema` (e.g., `repoUrl`, `depth`, `verbose`).

```yaml
invocation:
  cli:
    command: "git clone {repoUrl} {depth} {verbose}"
    templateVariables:
      repoUrl:
        property: "repoUrl"
      depth:
        property: "depth"
        format: "--depth {depth}"
      verbose:
        property: "verbose"
        format: "--verbose"
        omitIfFalse: true
```

## 7. Security Configuration Examples

### 7.1. TLS Configuration

To enable HTTPS for your MCP server, configure TLS in the `streamableHttpConfig`:

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
```

### 7.2. OAuth 2.0 Configuration

To protect your MCP server with OAuth 2.0 authentication:

```yaml
mcpFileVersion: "0.1.0"
name: protected-server
version: "1.0.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
    auth:
      authorizationServers:
        - https://auth.example.com
        - https://keycloak.company.com/auth/realms/mcp
      jwksUri: https://auth.example.com/.well-known/jwks.json
tools:
  - name: admin_tool
    description: "Administrative tool requiring elevated permissions"
    requiredScopes:
      - admin:write
      - users:manage
    # ... rest of tool definition
```

### 7.3. Combined TLS and OAuth Configuration

For maximum security, combine both TLS and OAuth:

```yaml
mcpFileVersion: "0.1.0"
name: secure-protected-server
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
```

## 8. Complete Example for a CLI

```yaml
mcpFileVersion: "0.1.0"
name: git-tools
version: "1.0.0"
runtime:
  transportProtocol: stdio
tools:
  - name: clone_repo
    title: "Clone Git Repository"
    description: "Clones a git repository from a URL to the local machine."
    inputSchema:
      type: object
      properties:
        repoUrl:
          type: string
          description: "The git URL of the repo to clone."
        depth:
          type: integer
          description: "The number of commits to clone."
        verbose:
          type: boolean
          description: "Whether to return verbose logs."
      required:
      - repoUrl
    invocation:
      cli:
        command: "git clone {repoUrl} {depth} {verbose}"
        templateVariables:
          repoUrl:
            property: "repoUrl"
          depth:
            property: "depth"
            format: "--depth {depth}"
          verbose:
            property: "verbose"
            format: "--verbose"
            omitIfFalse: true
```

## 9. Complete Example for an HTTP Server

```yaml
mcpFileVersion: "0.1.0"
name: user-service
version: "2.1.0"
runtime:
transportProtocol: streamablehttp
streamableHttpConfig:
  port: 3000
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
