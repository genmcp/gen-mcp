# MCP File Format Specification

## 1. Introduction

The MCP (Model Context Protocol) file is a YAML-based configuration that defines the capabilities of one or more MCP servers. It specifies the tools available, their input and output schemas, and how they should be invoked. This document details version `0.0.1` of the file format.

## 2. Top-Level Object

The root of the configuration is a single top-level object with the following fields:

| Field | Type | Description | Required |
|---|---|---|---|
| `mcpFileVersion` | string | The version of the MCP file format. Must be `"0.0.1"`. | Yes |
| `servers` | array of `MCPServer` | A list of servers defined in this file. | No |

### Example

```yaml
mcpFileVersion: 0.0.1
servers:
  # ... server definitions
```

## 3. MCPServer Object

An `MCPServer` object defines a single logical server and its set of tools.

| Field | Type | Description | Required |
|---|---|---|---|
| `name` | string | The name of the server. | Yes |
| `version` | string | The semantic version of the server's toolset. | Yes |
| `tools` | array of `Tool` | The tools provided by this server. | No |

### Example

```yaml
- name: my-awesome-server
  version: 1.2.3
  tools:
    # ... tool definitions
```

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

## 7. Complete Example

```yaml
mcpFileVersion: "0.0.1"
servers:
- name: git-tools
  version: "1.0.0"
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

- name: user-service
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
