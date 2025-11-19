---
layout: page
title: MCP File Format Specification
description: Complete reference guide for the GenMCP file format
---

# MCP File Format Specification

## 1. Introduction

GenMCP uses **two separate YAML configuration files** to define an MCP server:

1. **Tool Definitions File** - Defines the capabilities (tools, prompts, resources, resource templates) and invocation bases
2. **Server Config File** - Defines the server runtime configuration (transport protocol, logging, authentication, TLS)

This separation allows you to:
- Share tool definitions across different server configurations
- Version tool definitions and server configuration independently
- Deploy the same tools with different runtime settings (dev, staging, production)

Both files use schema version `0.2.0` and must be provided when running an MCP server with the `genmcp run` command.

## 2. Tool Definitions File

The Tool Definitions File defines what capabilities your MCP server provides. It contains tools, prompts, resources, resource templates, and invocation bases.

### 2.1. Top-Level Object

The root of the Tool Definitions File has the following structure:

| Field | Type | Description | Required |
|---|---|---|---|
| `kind` | string | Must be `"MCPToolDefinitions"`. | Yes |
| `schemaVersion` | string | The version of the MCP file format. Must be `"0.2.0"`. | Yes |
| `name` | string | The name of the server. | Yes |
| `version` | string | The semantic version of the server's toolset. | Yes |
| `instructions` | string | A set of instructions provided by the server to the client about how to use the server. | No |
| `invocationBases` | object | A set of reusable base configurations for invocations. Each key is a unique identifier, and each value is an invocation configuration (either `http` or `cli`). See [Section 6.3](#63-invocation-bases) for details. | No |
| `tools` | array of `Tool` | The tools provided by this server. | No |
| `prompts` | array of `Prompt` | The prompts provided by this server. | No |
| `resources` | array of `Resource` | The resources provided by this server. | No |
| `resourceTemplates` | array of `ResourceTemplate` | The resource templates provided by this server. | No |

### Example: Tool Definitions File

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: my-awesome-server
version: 1.2.3
instructions: |
  To clone and analyze a repository:
  1. First use clone_repo to clone the repository locally
  2. Then use get_commit_history to analyze the repository's history
  3. Use get_file_contents to examine specific files
  4. Finally, use generate_report to create a summary
invocationBases:
  baseGitApi:
    http:
      method: GET
      url: https://api.github.com/repos/{owner}/{repo}
tools:
  - name: clone_repo
    description: "Clones a git repository"
    inputSchema:
      type: object
      properties:
        repoUrl:
          type: string
      required:
        - repoUrl
    invocation:
      cli:
        command: "git clone {repoUrl}"
```

## 3. Server Config File

The Server Config File defines how the MCP server runs, including transport protocol, logging, authentication, and TLS settings.

### 3.1. Top-Level Object

The root of the Server Config File has the following structure:

| Field             | Type            | Description                                                                                                 | Required |
|-------------------|-----------------|-------------------------------------------------------------------------------------------------------------|----------|
| `kind`            | string          | Must be `"MCPServerConfig"`.                                                                                | Yes      |
| `schemaVersion`   | string          | The version of the MCP file format. Must be `"0.2.0"`.                                                      | Yes      |
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

## 4. ServerRuntime Object

The `ServerRuntime` object specifies the transport protocol and its configuration for the server.

| Field                  | Type                   | Description                                                                                                     | Required |
|------------------------|------------------------|-----------------------------------------------------------------------------------------------------------------|----------|
| `transportProtocol`    | string                 | The transport protocol to use. Must be one of `streamablehttp` or `stdio`.                                      | Yes      |
| `streamableHttpConfig` | `StreamableHTTPConfig` | Configuration for the `streamablehttp` transport protocol. Required if `transportProtocol` is `streamablehttp`. | No       |
| `stdioConfig`          | `StdioConfig`          | Configuration for the `stdio` transport protocol. Required if `transportProtocol` is `stdio`.                   | No       |
| `loggingConfig`        | `LoggingConfig`        | Configuration for server logging.                                                                               | No       |

### 4.1. StreamableHTTPConfig Object

| Field       | Type         | Description                                                    | Required |
|-------------|--------------|----------------------------------------------------------------|----------|
| `port`      | integer      | The port for the server to listen on.                          | Yes      |
| `basePath`  | string       | The base path for the MCP server. Defaults to `/mcp`.          | No       |
| `stateless` | boolean      | Indicates whether the server is stateless. Defaults to `true`. | No       |
| `auth`      | `AuthConfig` | OAuth 2.0 configuration for protected resources.               | No       |
| `tls`       | `TLSConfig`  | TLS configuration for HTTPS.                                   | No       |

### 4.2. TLSConfig Object

| Field      | Type   | Description                                                                                                      | Required |
|------------|--------|------------------------------------------------------------------------------------------------------------------|----------|
| `certFile` | string | The absolute path to the server's public certificate file on the runtime host where the MCP server will execute. | Yes      |
| `keyFile`  | string | The absolute path to the server's private key file on the runtime host where the MCP server will execute.        | Yes      |

### 4.3. AuthConfig Object

| Field                  | Type            | Description                                                                                                                                                                             | Required |
|------------------------|-----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| `authorizationServers` | array of string | List of authorization server URLs for OAuth 2.0 token validation.                                                                                                                       | No       |
| `jwksUri`              | string          | JSON Web Key Set URI for token signature verification. If no value is given but `authorizationServers` is set, gen-mcp will try to find a JWKS endpoint using different fallback paths. | No       |

### 4.4. StdioConfig Object

This object is currently empty and serves as a placeholder for future configuration options.

### 4.5. LoggingConfig Object

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

## 5. Primitive Objects

The MCP file format supports four types of primitive objects: Tools, Prompts, Resources, and Resource Templates. Each primitive object represents a capability that can be invoked by an MCP client. These are defined in the **Tool Definitions File**.

### 5.1. Tool Object

A `Tool` object describes a specific, invokable function.

| Field            | Type              | Description                                                                                              | Required |
|------------------|-------------------|----------------------------------------------------------------------------------------------------------|----------|
| `name`           | string            | A unique, programmatic identifier for the tool (e.g., `clone_repo`).                                     | Yes      |
| `title`          | string            | A human-readable title for display purposes (e.g., "Clone Git Repository").                              | No       |
| `description`    | string            | A detailed description of what the tool does, intended for an LLM to understand its function.            | Yes      |
| `inputSchema`    | `JsonSchema`      | A JSON Schema object defining the parameters the tool accepts.                                           | Yes      |
| `outputSchema`   | `JsonSchema`      | A JSON Schema object defining the structure of the tool's output.                                        | No       |
| `invocation`     | `Invocation`      | An object describing how to execute the tool. Can be `http`, `cli`, or `extends`.                        | Yes      |
| `requiredScopes` | array of string   | OAuth 2.0 scopes required to execute this tool. Only relevant when the server uses OAuth authentication. | No       |
| `annotations`    | `ToolAnnotations` | Annotations to indicate tool behaviour to the client.                                                    | No       |

#### 5.1.1. ToolAnnotations Object

| Field             | Type    | Description                                                                                                                       | Required |
|-------------------|---------|-----------------------------------------------------------------------------------------------------------------------------------|----------|
| `destructiveHint` | boolean | If true, the tool may perform destructive updates to its environment. If false, the tool performs only additive updates.          | No       |
| `idempotentHint`  | boolean | If true, calling the tool repeatedly with the same arguments will have no additional effect on its environment.                   | No       |
| `openWorldHint`   | boolean | If true, this tool may interact with an "open world" or external entities. If false, this tool's domain of interaction is closed. | No       |
| `readOnlyHint`    | boolean | If true, the tool does not modify its environment.                                                                                | No       |

### 5.2. Prompt Object

A `Prompt` object describes a natural-language or LLM-style function invocation.

| Field            | Type                      | Description                                                                                                | Required |
|------------------|---------------------------|------------------------------------------------------------------------------------------------------------|----------|
| `name`           | string                    | A unique, programmatic identifier for the prompt.                                                          | Yes      |
| `title`          | string                    | A human-readable title for display purposes.                                                               | No       |
| `description`    | string                    | A detailed description of what the prompt does.                                                            | Yes      |
| `arguments`      | array of `PromptArgument` | List of template arguments for the prompt.                                                                 | No       |
| `inputSchema`    | `JsonSchema`              | A JSON Schema object defining the parameters the prompt accepts.                                           | Yes      |
| `outputSchema`   | `JsonSchema`              | A JSON Schema object defining the structure of the prompt's output.                                        | No       |
| `invocation`     | `Invocation`              | An object describing how to execute the prompt. Can be `http`, `cli`, or `extends`.                        | Yes      |
| `requiredScopes` | array of string           | OAuth 2.0 scopes required to execute this prompt. Only relevant when the server uses OAuth authentication. | No       |

#### 5.2.1. PromptArgument Object

| Field         | Type    | Description                             | Required |
|---------------|---------|-----------------------------------------|----------|
| `name`        | string  | Unique identifier for the argument.     | Yes      |
| `title`       | string  | Human-readable title for display.       | No       |
| `description` | string  | Detailed explanation of the argument.   | No       |
| `required`    | boolean | Indicates if the argument is mandatory. | No       |

### 5.3. Resource Object

A `Resource` object represents a retrievable or executable resource.

| Field            | Type            | Description                                                                                                 | Required |
|------------------|-----------------|-------------------------------------------------------------------------------------------------------------|----------|
| `name`           | string          | A unique, programmatic identifier for the resource.                                                         | Yes      |
| `title`          | string          | A human-readable title for display purposes.                                                                | No       |
| `description`    | string          | A detailed description of the resource.                                                                     | Yes      |
| `mimeType`       | string          | The MIME type of this resource, if known.                                                                   | No       |
| `size`           | integer         | The size of the raw resource content in bytes, if known.                                                    | No       |
| `uri`            | string          | The URI of this resource.                                                                                   | Yes      |
| `inputSchema`    | `JsonSchema`    | A JSON Schema object defining the parameters the resource accepts. Optional for resources without inputs.   | No       |
| `outputSchema`   | `JsonSchema`    | A JSON Schema object defining the structure of the resource's output.                                       | No       |
| `invocation`     | `Invocation`    | An object describing how to invoke the resource. Can be `http`, `cli`, or `extends`.                        | Yes      |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to access this resource. Only relevant when the server uses OAuth authentication. | No       |

### 5.4. ResourceTemplate Object

A `ResourceTemplate` object represents a reusable URI-based template for resources.

| Field            | Type            | Description                                                                                                          | Required |
|------------------|-----------------|----------------------------------------------------------------------------------------------------------------------|----------|
| `name`           | string          | A unique, programmatic identifier for the resource template.                                                         | Yes      |
| `title`          | string          | A human-readable title for display purposes.                                                                         | No       |
| `description`    | string          | A detailed description of the resource template.                                                                     | Yes      |
| `mimeType`       | string          | MIME type for resources matching this template.                                                                      | No       |
| `uriTemplate`    | string          | URI template (RFC 6570) used to construct resource URIs.                                                             | Yes      |
| `inputSchema`    | `JsonSchema`    | A JSON Schema object defining the parameters the resource template accepts.                                          | Yes      |
| `outputSchema`   | `JsonSchema`    | A JSON Schema object defining the structure of the resource template's output.                                       | No       |
| `invocation`     | `Invocation`    | An object describing how to invoke the resource template. Can be `http`, `cli`, or `extends`.                        | Yes      |
| `requiredScopes` | array of string | OAuth 2.0 scopes required to access this resource template. Only relevant when the server uses OAuth authentication. | No       |

## 6. JsonSchema Object

The `inputSchema` and `outputSchema` fields use the JSON Schema standard to define data structures.

| Field                  | Type                    | Description                                                                                   |
|------------------------|-------------------------|-----------------------------------------------------------------------------------------------|
| `type`                 | string                  | The data type. Can be `string`, `number`, `integer`, `boolean`, `array`, `object`, or `null`. |
| `description`          | string                  | A human-readable description of the schema or field.                                          |
| `properties`           | map[string]`JsonSchema` | For `object` types, defines the named properties of the object.                               |
| `required`             | array of string         | For `object` types, lists the property names that are required.                               |
| `items`                | `JsonSchema`            | For `array` types, defines the schema of each item in the array.                              |
| `additionalProperties` | boolean                 | For `object` types, specifies whether additional properties are allowed.                      |

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

## 7. Invocation Object

The `invocation` object specifies how a tool, prompt, resource, or resource template is executed. It must contain exactly one of the following types: `http`, `cli`, or `extends`.

### 7.1. HTTP Invocation

The `http` invocation type is used for tools that are called via an HTTP request.

| Field | Type | Description | Required |
|---|---|---|---|
| `method` | string | The HTTP method (e.g., `GET`, `POST`). | Yes |
| `url` | string | The URL to send the request to. It can be a template. Input parameters from the `inputSchema` are substituted into placeholders like `{paramName}`. Can also use `{headers.HeaderName}` to access incoming HTTP headers (streamablehttp only) or `${ENV_VAR_NAME}` / `{env.ENV_VAR_NAME}` for environment variables. | Yes |
| `headers` | map[string]string | HTTP headers to include in the request. Values can use the same templating as `url`, supporting `{paramName}` for input schema parameters, `{headers.HeaderName}` for incoming headers (streamablehttp only), and `${ENV_VAR_NAME}` / `{env.ENV_VAR_NAME}` for environment variables. | No |

#### Example: Basic Usage

```yaml
invocation:
  http:
    method: GET
    url: http://localhost:8080/users/{userId}
```

#### Example: With Static Headers

```yaml
invocation:
  http:
    method: POST
    url: http://localhost:8080/api/data
    headers:
      Authorization: Bearer secret-token
      X-Custom-Header: static-value
```

#### Example: With Template Headers

```yaml
invocation:
  http:
    method: POST
    url: http://localhost:8080/users
    headers:
      X-User-Id: "{userId}"
      X-Tenant: "{tenant}"
```

#### Example: Forwarding Incoming Headers (streamablehttp only)

```yaml
invocation:
  http:
    method: GET
    url: http://localhost:8080/proxy
    headers:
      Authorization: "{headers.Authorization}"
      X-Request-Id: "{headers.X-Request-Id}"
```

#### Example: Using Headers in URL Template (streamablehttp only)

```yaml
invocation:
  http:
    method: GET
    url: http://localhost:8080/users/{headers.X-User-Id}
```

### 7.2. CLI Invocation

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

### 7.3. Invocation Bases

Invocation bases allow you to define reusable configurations that can be referenced by multiple tools, prompts, resources, or resource templates. This reduces duplication and makes it easier to maintain consistent configuration across primitives.

The `invocationBases` field is defined in the **Tool Definitions File**. It contains named base configurations:

```yaml
invocationBases:
  baseHttpConfig:
    http:
      method: GET
      url: https://api.example.com/base
  baseCLIConfig:
    cli:
      command: "git {operation}"
      templateVariables:
        operation:
          format: "{operation}"
```

Each key in `invocationBases` is a unique identifier for the base configuration, and the value is an invocation configuration (either `http` or `cli`).

### 7.4. Extends Invocation

The `extends` invocation type allows you to reference and modify a base configuration defined in `invocationBases`. This provides a powerful way to compose configurations by reusing common settings and making targeted modifications.

| Field      | Type   | Description                                                                                                                                                 | Required |
|------------|--------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| `from`     | string | The identifier of the base configuration in `invocationBases` to extend.                                                                                    | Yes      |
| `extend`   | object | Fields to add or merge with the base configuration. For strings, values are concatenated. For maps, keys are merged. For arrays, items are appended.        | No       |
| `override` | object | Fields to completely replace in the base configuration.                                                                                                     | No       |
| `remove`   | object | Fields to remove from the base configuration. For maps, specify keys to remove. For arrays, specify values to remove. For strings, sets the field to empty. | No       |

#### Operations Behavior

- **extend**: Merges new values with existing ones
  - Strings: Concatenates values (e.g., `"hello"` + `" world"` = `"hello world"`)
  - Maps: Merges keys (new keys are added, existing keys are updated with new values)
  - Arrays: Appends items to the end

- **override**: Completely replaces values
  - Any field specified in `override` will replace the entire value from the base configuration
  - Zero values (empty strings, 0, false) are skipped

- **remove**: Removes values
  - Strings: Sets to empty string
  - Maps: Removes specified keys (specify keys as an array or as a map with empty values)
  - Arrays: Removes all occurrences of specified values

**Important**: You cannot use multiple operations (`extend`, `override`, `remove`) on the same field. Each field can only be modified by one operation.

#### Example: HTTP Extension

```yaml
invocationBases:
  baseUserApi:
    http:
      method: GET
      url: https://api.example.com/users

tools:
  - name: get_user_by_id
    description: "Get a specific user by ID"
    inputSchema:
      type: object
      properties:
        userId:
          type: string
      required:
        - userId
    invocation:
      extends:
        from: baseUserApi
        extend:
          url: "/{userId}"  # Concatenated to base URL

  - name: create_user
    description: "Create a new user"
    inputSchema:
      type: object
      properties:
        name:
          type: string
      required:
        - name
    invocation:
      extends:
        from: baseUserApi
        override:
          method: POST  # Changes GET to POST
```

#### Example: CLI Extension

```yaml
invocationBases:
  baseGitCommand:
    cli:
      command: "git {operation} {verbose}"
      templateVariables:
        verbose:
          format: "--verbose"
          omitIfFalse: true

tools:
  - name: git_clone
    description: "Clone a repository"
    inputSchema:
      type: object
      properties:
        repoUrl:
          type: string
        verbose:
          type: boolean
      required:
        - repoUrl
    invocation:
      extends:
        from: baseGitCommand
        extend:
          command: " {repoUrl}"  # Appends to base command
        override:
          templateVariables:
            operation:
              format: "clone"
```

#### Example: Remove Operation

```yaml
invocationBases:
  baseApiCall:
    http:
      method: GET
      url: https://api.example.com/{endpoint}

tools:
  - name: simple_call
    description: "Simple API call without endpoint parameter"
    inputSchema:
      type: object
    invocation:
      extends:
        from: baseApiCall
        remove:
          url: "{endpoint}"  # Removes the {endpoint} placeholder
        extend:
          url: "/simple"  # Adds the fixed endpoint
```

## 8. Complete Examples

### 8.1. Basic Example with Two Files

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: Feature Request API
version: "0.0.1"
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

### 8.2. Advanced Logging Configuration

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: Production API
version: "1.0.0"
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

### 8.3. Using Invocation Bases and Extends

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: user-management-api
version: "1.0.0"
invocationBases:
  # Base configuration for user API calls
  baseUserApi:
    http:
      method: GET
      url: https://api.example.com/v1/users

  # Base configuration for admin API calls
  baseAdminApi:
    http:
      method: GET
      url: https://api.example.com/v1/admin

tools:
  - name: list_users
    description: "List all users"
    inputSchema:
      type: object
    invocation:
      extends:
        from: baseUserApi
        # No modifications needed - uses base as-is

  - name: get_user
    description: "Get a specific user by ID"
    inputSchema:
      type: object
      properties:
        userId:
          type: string
      required:
        - userId
    invocation:
      extends:
        from: baseUserApi
        extend:
          url: "/{userId}"  # Results in: https://api.example.com/v1/users/{userId}

  - name: create_user
    description: "Create a new user"
    inputSchema:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
      required:
        - name
        - email
    invocation:
      extends:
        from: baseUserApi
        override:
          method: POST  # Changes from GET to POST

  - name: delete_user
    description: "Delete a user by ID"
    inputSchema:
      type: object
      properties:
        userId:
          type: string
      required:
        - userId
    invocation:
      extends:
        from: baseUserApi
        extend:
          url: "/{userId}"
        override:
          method: DELETE  # Changes from GET to DELETE

  - name: get_admin_stats
    description: "Get administrative statistics"
    inputSchema:
      type: object
    invocation:
      extends:
        from: baseAdminApi
        extend:
          url: "/stats"  # Results in: https://api.example.com/v1/admin/stats
    requiredScopes:
      - admin:read
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

## 9. Security Configuration Examples

### 9.1. TLS Configuration

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: secure-server
version: "1.0.0"
tools:
  - name: secure_endpoint
    description: "A secure endpoint"
    inputSchema:
      type: object
    invocation:
      http:
        method: GET
        url: https://api.example.com/secure
```

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

### 9.2. OAuth 2.0 Configuration

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: protected-server
version: "1.0.0"
tools:
  - name: admin_tool
    description: "Administrative tool requiring elevated permissions"
    requiredScopes:
      - admin:write
      - users:manage
    inputSchema:
      type: object
    invocation:
      http:
        method: POST
        url: https://api.example.com/admin/action
```

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

### 9.3. Combined TLS and OAuth Configuration

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: secure-protected-server
version: "1.0.0"
tools:
  - name: secure_admin_tool
    description: "Secure administrative tool"
    requiredScopes:
      - admin:write
    inputSchema:
      type: object
    invocation:
      http:
        method: POST
        url: https://api.example.com/admin/secure-action
```

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

## 10. Complete Example for a CLI

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: git-tools
version: "1.0.0"
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

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: stdio
```

## 11. Complete Example for an HTTP Server

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: user-service
version: "2.1.0"
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

**Server Config File** (`mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 3000
```
