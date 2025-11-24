# JSON Schema Generator for MCP Files

This utility is a simple Go program responsible for generating JSON schemas from the MCP file structs defined in the main project.

It ensures that any MCP configuration files conform to the expected format, providing a reliable way to validate data integrity.

## How It Works

The generator uses the [`github.com/invopop/jsonschema`](https://github.com/invopop/jsonschema) library to reflect the Go structs for both MCP file types:
- `MCPToolDefinitionsFile` from `pkg/config/definitions`
- `MCPServerConfigFile` from `pkg/config/server`

It inspects the structs' fields, types, and tags to generate corresponding JSON Schemas.  
The resulting schemas are then written to the `specs` directory in two forms for each file type:

- A **versioned** schema file, based on the current `SchemaVersion` (0.2.0)
- The **latest** schema file, which always contains the same content as the versioned file

## Usage

To generate or update the JSON schemas:

1. Navigate to the directory containing the utility:
    ```bash
    cd hack/jsonschemagen
    ```

2. Run the program:
    ```bash
    go run main.go
    ```

This will generate the schema files inside the `specs` directory relative to the project root.

## Output

The tool generates four files (two for each file type):

### Tool Definitions Schema
- **`../../specs/mcpfile-schema-<version>.json`** — The versioned JSON schema for MCP files, where `<version>` is taken from `definitions.SchemaVersion`.
- **`../../specs/mcpfile-schema.json`** — The "latest" schema, identical in content to the versioned one.

### Server Config Schema
- **`../../specs/mcpserver-schema-<version>.json`** — The versioned JSON schema for server config files, where `<version>` is taken from `definitions.SchemaVersion`.
- **`../../specs/mcpserver-schema.json`** — The "latest" schema, identical in content to the versioned one.

Example output:

```
specs/
├── mcpfile-schema-0.2.0.json
├── mcpfile-schema.json
├── mcpserver-schema-0.2.0.json
└── mcpserver-schema.json
```

## Using the Generated Schemas

The schemas can be referenced in YAML files using the `yaml-language-server` directive:

**MCP File:**
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpfile-schema-0.2.0.json

kind: MCPToolDefinitions
schemaVersion: "0.2.0"
# ... rest of file
```

**Server Config File:**
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpserver-schema-0.2.0.json

kind: MCPServerConfig
schemaVersion: "0.2.0"
# ... rest of file
```

Documentation about how to use the generated schemas is available in the [root README](../../README.md#-authoring-mcpfileyaml-with-auto-complete).
