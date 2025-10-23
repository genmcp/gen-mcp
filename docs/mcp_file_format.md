# MCP File Format Specification

## 1. Introduction

The MCP (Model Context Protocol) file format consists of two YAML-based configuration files that define an MCP server:
- **mcpserver.yaml**: Contains the server runtime configuration (transport protocol, port, logging, etc.)
- **mcpfile.yaml**: Contains the tool, prompt, and resource definitions

Both files include a `kind` field (similar to Kubernetes resources) to identify their purpose. This document details version `0.1.0` of the file format.

## 2. File Types

### 2.1. MCPServerConfig (mcpserver.yaml)

The server configuration file contains runtime settings for the MCP server.

| Field | Type | Description | Required |
|---|---|---|---|
| `kind` | string | Must be `"MCPServerConfig"`. | Yes |
| `schemaVersion` | string | The version of the MCP file format. Must be `"0.1.0"`. | Yes |
| `name` | string | The name of the server. | Yes |
| `version` | string | The semantic version of the server's toolset. | Yes |
| `runtime` | `ServerRuntime` | The runtime settings for the server. | Yes |

#### Example

```yaml
kind: MCPServerConfig
schemaVersion: 0.1.0
name: my-awesome-server
version: 1.2.3
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
```

### 2.2. MCPToolDefinitions (mcpfile.yaml)

The tool definitions file contains the tools, prompts, and resources provided by the server.

| Field | Type | Description | Required |
|---|---|---|---|
| `kind` | string | Must be `"MCPToolDefinitions"`. | Yes |
| `schemaVersion` | string | The version of the MCP file format. Must be `"0.1.0"`. | Yes |
| `instructions` | string | A set of instructions provided by the server to the client about how to use the server. | No |
| `tools` | array of `Tool` | The tools provided by this server. | No |
| `prompts` | array of `Prompt` | The prompts provided by this server. | No |
| `resources` | array of `Resource` | The resources provided by this server. | No |
| `resourceTemplates` | array of `ResourceTemplate` | The resource templates provided by this server. | No |

#### Example

```yaml
kind: MCPToolDefinitions
schemaVersion: 0.1.0
instructions: |
  To clone and analyze a repository:
  1. First use clone_repo to clone the repository locally
  2. Then use get_commit_history to analyze the repository's history
  3. Use get_file_contents to examine specific files
  4. Finally, use generate_report to create a summary
tools:
  # ... tool definitions
```

### 2.3. Legacy Single-File Format (Deprecated)

For backward compatibility, the legacy single-file format is still supported but is deprecated. In the legacy format, both server configuration and tool definitions are in a single `mcpfile.yaml` file without a `kind` field. New projects should use the separate file format described above.

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

## 4. Primitive Objects

The MCP file format supports four types of primitive objects: Tools, Prompts, Resources, and Resource Templates. Each primitive object represents a capability that can be invoked by an MCP client.

### 4.1. Tool Object

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

### 4.2. Prompt Object

A `Prompt` object describes a natural-language or LLM-style function invocation.

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | A unique, programmatic identifier for the prompt. | Yes |
| `title` | string | A human-readable title for display purposes. | No |
| `description` | string | A detailed description of what the prompt does. | Yes |
| `arguments` | array of `PromptArgument` | List of template arguments for the prompt. | No |
| `inputSchema` | `JsonSchema` | A JSON Schema object defining the parameters the prompt accepts. | Yes |
| `outputSchema` | `JsonSchema` | A JSON Schema object defining the structure of the prompt's output. | No |
| `invocation` | `Invocation` | An object describing how to execute the prompt. Must contain a single key: either `http` or `cli`. | Yes |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to execute this prompt. Only relevant when the server uses OAuth authentication. | No |

#### 4.2.1. PromptArgument Object

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | Unique identifier for the argument. | Yes |
| `title` | string | Human-readable title for display. | No |
| `description` | string | Detailed explanation of the argument. | No |
| `required` | boolean | Indicates if the argument is mandatory. | No |

### 4.3. Resource Object

A `Resource` object represents a retrievable or executable resource.

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | A unique, programmatic identifier for the resource. | Yes |
| `title` | string | A human-readable title for display purposes. | No |
| `description` | string | A detailed description of the resource. | Yes |
| `mimeType` | string | The MIME type of this resource, if known. | No |
| `size` | integer | The size of the raw resource content in bytes, if known. | No |
| `uri` | string | The URI of this resource. | Yes |
| `inputSchema` | `JsonSchema` | A JSON Schema object defining the parameters the resource accepts. | Yes |
| `outputSchema` | `JsonSchema` | A JSON Schema object defining the structure of the resource's output. | No |
| `invocation` | `Invocation` | An object describing how to invoke the resource. Must contain a single key: either `http` or `cli`. | Yes |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to access this resource. Only relevant when the server uses OAuth authentication. | No |

### 4.4. ResourceTemplate Object

A `ResourceTemplate` object represents a reusable URI-based template for resources.

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | A unique, programmatic identifier for the resource template. | Yes |
| `title` | string | A human-readable title for display purposes. | No |
| `description` | string | A detailed description of the resource template. | Yes |
| `mimeType` | string | MIME type for resources matching this template. | No |
| `uriTemplate` | string | URI template (RFC 6570) used to construct resource URIs. | Yes |
| `inputSchema` | `JsonSchema` | A JSON Schema object defining the parameters the resource template accepts. | Yes |
| `outputSchema` | `JsonSchema` | A JSON Schema object defining the structure of the resource template's output. | No |
| `invocation` | `Invocation` | An object describing how to invoke the resource template. Must contain a single key: either `http` or `cli`. | Yes |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to access this resource template. Only relevant when the server uses OAuth authentication. | No |

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
| `format` | string | A format string for the argument (e.g., `--depth {depth}`). The placeholder within this string is replaced by the actual value of the input property. | Yes |
| `omitIfFalse` | boolean | If `true`, the entire formatted argument is omitted from the command if the input property's value is `false`. Defaults to `false`. | No |

#### Example

In this example, the `{repoUrl}`, `{depth}`, and `{verbose}` placeholders in `command` are defined in `templateVariables`. The key name corresponds to the property in the `inputSchema`.

```yaml
invocation:
  cli:
    command: "git clone {repoUrl} {depth} {verbose}"
    templateVariables:
      depth:
        format: "--depth {depth}"
      verbose:
        format: "--verbose"
        omitIfFalse: true
```

## 7. Complete Examples

### 7.1. Basic Server with Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
name: Feature Request API
version: "0.0.1"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8008
  loggingConfig:
    enableMcpLogs: true
    encoding: "console"
    level: "debug"
```

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
tools:
  - name: get_features
    title: "Get all features"
    description: "Returns a list of all features sorted by upvotes (highest first)"
    inputSchema:
      type: object
    invocation:
      http:
        method: GET
        url: http://localhost:9090/features
```

### 7.2. Production Server with Advanced Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
name: Production API
version: "1.0.0"
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

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
tools:
  - name: health_check
    description: "Returns the health status of the service"
    inputSchema:
      type: object
    invocation:
      http:
        method: GET
        url: http://localhost:8080/health
```

### 7.3. Server with Disabled MCP Logging

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
name: Silent API
version: "1.0.0"
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

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
tools:
  - name: process_data
    description: "Processes data without sending logs to MCP clients"
    inputSchema:
      type: object
    invocation:
      http:
        method: POST
        url: http://localhost:8080/process
```

## 8. Security Configuration Examples

### 8.1. TLS Configuration

To enable HTTPS for your MCP server, configure TLS in the `streamableHttpConfig`:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
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

### 8.2. OAuth 2.0 Configuration

To protect your MCP server with OAuth 2.0 authentication:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
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
```

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
tools:
  - name: admin_tool
    description: "Administrative tool requiring elevated permissions"
    requiredScopes:
      - admin:write
      - users:manage
    # ... rest of tool definition
```

### 8.3. Combined TLS and OAuth Configuration

For maximum security, combine both TLS and OAuth:

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
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

## 9. Complete Example for a CLI Server

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
name: git-tools
version: "1.0.0"
runtime:
  transportProtocol: stdio
```

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
instructions: |
  This server provides Git repository management tools. For typical workflows:
  1. Use clone_repo to get a local copy of a repository
  2. Use check_status to verify the repository state
  3. Use commit_changes to save modifications

  For shallow clones, always specify a depth parameter to save bandwidth.
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
          depth:
            format: "--depth {depth}"
          verbose:
            format: "--verbose"
            omitIfFalse: true
```

## 10. Complete Example for an HTTP Server

**mcpserver.yaml:**
```yaml
kind: MCPServerConfig
schemaVersion: "0.1.0"
name: user-service
version: "2.1.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 3000
```

**mcpfile.yaml:**
```yaml
kind: MCPToolDefinitions
schemaVersion: "0.1.0"
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
