# JSON Schema Generator for MCPFile

This utility is a simple Go program responsible for generating a JSON schema from the `MCPFile` struct defined in the main project.

It ensures that any `mcpfile` conforms to the expected format, providing a reliable way to validate data integrity.

## How It Works

The generator uses the [`github.com/invopop/jsonschema`](https://github.com/invopop/jsonschema) library to reflect the `mcpfile.MCPFile` Go struct.  
It inspects the struct's fields, types, and tags to generate a corresponding JSON Schema.  
The resulting schema is then written to the `specs` directory in two forms:

- A **versioned** schema file, based on the current `mcpfile.MCPFileVersion`
- A **latest** schema file, which always contains the same content as the versioned file

## Usage

To generate or update the JSON schema:

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

The tool generates two files:

- **`../../specs/mcpfile-schema-<version>.json`** — The versioned JSON schema, where `<version>` is taken from `mcpfile.MCPFileVersion`.
- **`../../specs/mcpfile-schema.json`** — The "latest" schema, identical in content to the versioned one.

Example output:

```
specs/
├── mcpfile-schema-0.1.0.json
└── mcpfile-schema.json
```


## Using the Generated Schema

Documentation about how to use the generated schema is available in the [root README](../../README.md#-authoring-mcpfileyaml-with-auto-complete).
