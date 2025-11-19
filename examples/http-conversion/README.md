# HTTP Endpoint Conversion Example

📹 **[Watch the demo video](https://youtu.be/boMyFzpgJoA)** to see this example in action!

This example demonstrates gen-mcp's ability to automatically convert HTTP REST API endpoints into MCP tools. gen-mcp can expose any REST API as MCP tools that can be called by AI assistants, eliminating the need to write custom MCP server code.

## Getting Started

### 1. Start the Feature Request API Server

First, run the Go server that provides the REST API:

```bash
cd feature-requests
go run main.go
```

The API will be available at `http://localhost:9090` with endpoints:
- `GET /features` - List all features (summaries only, sorted by upvotes)
- `GET /features/top` - Get the highest-voted feature (summary only)
- `GET /features/{id}` - Get detailed information about a specific feature
- `POST /features` - Add a new feature request
- `POST /features/vote` - Vote for a feature (increases upvotes)
- `POST /features/complete` - Mark a feature request as completed
- `DELETE /features/{id}` - Delete a feature request
- `GET /openapi.json` - Get OpenAPI specification
- `POST /prompts/feature-analysis` - Generate feature analysis prompt
- `GET  /features/progress-report` - Get feature progress report (static resource)

### 2. Generate Initial MCP Configuration

Use gen-mcp to automatically generate a starter configuration from the API:

```bash
genmcp convert http://localhost:9090/openapi.json
```

This creates two files:
- `mcpfile.yaml` - Tool definitions (tools, prompts, resources)
- `mcpserver.yaml` - Server configuration (runtime settings)

### 3. Customize the Configuration

Edit the generated files to:
- **Tool Definitions File** (`mcpfile.yaml`): Select which endpoints should be exposed as MCP tools, improve tool descriptions, add usage instructions, configure input validation schemas
- **Server Config File** (`mcpserver.yaml`): Configure runtime settings like port, logging, authentication

Example customizations in this demo:
- Clear, specific descriptions for each tool
- Guidance on when to call related tools (e.g., "Always call get_features-id after this tool...")
- Proper input schemas with required parameters
- Only exposing read endpoints initially (GET operations) for safety

### 4. Start the MCP Server

Launch the gen-mcp server with both files:

```bash
   genmcp run -f mcpfile.yaml -s mcpserver.yaml
```

The MCP server will run on port 8080 (as configured) and expose the HTTP endpoints as MCP tools that AI assistants can call seamlessly.

## Key gen-mcp HTTP Conversion Features

- **Automatic Tool Generation**: HTTP endpoints become MCP tools automatically from OpenAPI specs
- **Path Parameter Substitution**: URL templates like `{id}` are handled automatically
- **Schema Validation**: Input parameters are validated before API calls
- **Streamable HTTP Protocol**: Real-time communication via `streamablehttp`
- **Flexible Configuration**: Full control over which endpoints to expose and how
- **POST/PUT/DELETE Support**: Can expose write operations like adding features, voting, completing, and deleting
