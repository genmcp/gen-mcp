# JSON Schema Generator for MCPFile

This utility is a simple Go program responsible for generating a JSON schema from the `MCPFile` struct defined in the main project.

This ensures that any `mcpfile` conforms to the expected format, providing a reliable way to validate data integrity.

## How It Works

The generator uses the `github.com/invopop/jsonschema` library to reflect the `mcpfile.MCPFile` Go struct. It inspects the struct's fields, types, and tags to generate a corresponding JSON schema. 
The resulting schema is then written to the `mcpfile-schema.json` file in the project's root directory.

## Prerequisites

- Go 1.24 or later installed on your system.

## Usage

To generate or update the JSON schema, follow these steps:

1.  Navigate to the directory containing the utility:
    ```bash
    cd hack/jsonschemagen
    ```

2.  Run the program:
    ```bash
    go run main.go
    ```

This command will execute the `main.go` file, which generates the schema and saves it to the root of the repository.

## Output

The tool generates a single file:

-   **`../../mcpfile-schema.json`**: The JSON schema definition for the `MCPFile` struct. This file is placed in the root directory of the project.

## Using the Generated Schema

This is documented in the [root README](../../README.md#-authoring-mcpfileyaml-with-auto-complete) file.
