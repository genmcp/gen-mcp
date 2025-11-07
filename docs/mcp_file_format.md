# MCP File Format Specification

## 1. Introduction

The MCP (Model Context Protocol) file is a YAML-based configuration that defines the capabilities of an MCP server. It specifies the tools available, their input and output schemas, and how they should be invoked. This document details version `0.1.0` of the file format.

## 2. Top-Level Object

The root of the configuration is a single top-level object with the following fields:

| Field | Type | Description | Required |
|---|---|---|---|
| `mcpFileVersion` | string | The version of the MCP file format. Must be `"0.1.0"`. | Yes |
| `name` | string | The name of the server. | Yes |
| `version` | string | The semantic version of the server's toolset. | Yes |
| `runtime` | `ServerRuntime` | The runtime settings for the server. If omitted, it defaults to `streamablehttp` on port `3000`. | No |
| `instructions` | string | A set of instructions provided by the server to the client about how to use the server. | No |
| `invocationBases` | object | A set of reusable base configurations for invocations. Each key is a unique identifier, and each value is an invocation configuration (either `http` or `cli`). See [Section 6.3](#63-invocation-bases) for details. | No |
| `tools` | array of `Tool` | The tools provided by this server. | No |
| `prompts` | array of `Prompt` | The prompts provided by this server. | No |
| `resources` | array of `Resource` | The resources provided by this server. | No |
| `resourceTemplates` | array of `ResourceTemplate` | The resource templates provided by this server. | No |

### Example

```yaml
mcpFileVersion: 0.1.0
name: my-awesome-server
version: 1.2.3
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080
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
  # ... tool definitions
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
| `invocation` | `Invocation` | An object describing how to execute the tool. Can be `http`, `cli`, or `extends`. | Yes |
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
| `invocation` | `Invocation` | An object describing how to execute the prompt. Can be `http`, `cli`, or `extends`. | Yes |
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
| `invocation` | `Invocation` | An object describing how to invoke the resource. Can be `http`, `cli`, or `extends`. | Yes |
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
| `invocation` | `Invocation` | An object describing how to invoke the resource template. Can be `http`, `cli`, or `extends`. | Yes |
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

The `invocation` object specifies how a tool, prompt, resource, or resource template is executed. It must contain exactly one of the following types: `http`, `cli`, or `extends`.

### 6.1. HTTP Invocation

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

### 6.3. Invocation Bases

Invocation bases allow you to define reusable configurations that can be referenced by multiple tools, prompts, resources, or resource templates. This reduces duplication and makes it easier to maintain consistent configuration across primitives.

The `invocationBases` field is defined at the top level of the MCP file and contains named base configurations:

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

### 6.4. Extends Invocation

The `extends` invocation type allows you to reference and modify a base configuration defined in `invocationBases`. This provides a powerful way to compose configurations by reusing common settings and making targeted modifications.

| Field | Type | Description | Required |
|---|---|---|---|
| `from` | string | The identifier of the base configuration in `invocationBases` to extend. | Yes |
| `extend` | object | Fields to add or merge with the base configuration. For strings, values are concatenated. For maps, keys are merged. For arrays, items are appended. | No |
| `override` | object | Fields to completely replace in the base configuration. | No |
| `remove` | object | Fields to remove from the base configuration. For maps, specify keys to remove. For arrays, specify values to remove. For strings, sets the field to empty. | No |

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

## 7. Complete Examples

### 7.1. Basic Logging Configuration

```yaml
mcpFileVersion: "0.1.0"
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

### 7.2. Advanced Logging Configuration

```yaml
mcpFileVersion: "0.1.0"
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

### 7.3. Disabled MCP Logging

```yaml
mcpFileVersion: "0.1.0"
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

### 7.4. Using Invocation Bases and Extends

This example demonstrates how to use `invocationBases` and the `extends` invocation type to reduce configuration duplication:

```yaml
mcpFileVersion: "0.1.0"
name: user-management-api
version: "1.0.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8080

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

## 8. Security Configuration Examples

### 8.1. TLS Configuration

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

### 8.2. OAuth 2.0 Configuration

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

### 8.3. Combined TLS and OAuth Configuration

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

## 9. Complete Example for a CLI

```yaml
mcpFileVersion: "0.1.0"
name: git-tools
version: "1.0.0"
runtime:
  transportProtocol: stdio
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
