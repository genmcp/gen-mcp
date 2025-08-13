# HTTP Endpoint Conversion on OpenShift Example

📹 **[Watch the demo video](https://youtu.be/boMyFzpgJoA)** to see this example in action!

This example demonstrates AutoMCP's ability to automatically convert HTTP REST API endpoints into MCP tools. AutoMCP can expose any REST API as MCP tools that can be called by AI assistants, eliminating the need to write custom MCP server code.

## Getting Started

### 1. Deploy the Feature Request API Server

First, deploy the feature request server to your cluster, and create the associated service and route:

```bash
cd feature-requests
ko apply -f config/deployment.yaml -f config/service.yaml -f config/route.yaml
```

The API will be available at the url associated with your route with endpoints:
- `GET /features` - List all features (summaries only, sorted by upvotes)
- `GET /features/top` - Get the highest-voted feature (summary only)
- `GET /features/{id}` - Get detailed information about a specific feature
- `POST /features` - Add a new feature request
- `POST /features/vote` - Vote for a feature (increases upvotes)
- `POST /features/complete` - Mark a feature request as completed
- `DELETE /features/{id}` - Delete a feature request
- `GET /openapi.json` - Get OpenAPI specification

### 2. Generate Initial MCP Configuration

Use AutoMCP to automatically generate a starter configuration from the API:

```bash
automcp convert <base route url>/openapi.json -H <base route url>
```

Note: we are using the `-H` flag here to set the base host url for the api spec, as the openapi.json file says that the endpoints are available at `localhost:9090`.

This creates an initial `mcpfile.yaml` based on the OpenAPI specification, with the endpoints all pointing to our OpenShift route.

### 3. Customize the Configuration

Edit the generated `mcpfile.yaml` to:
- Select which endpoints should be exposed as MCP tools
- Improve tool descriptions to help AI models understand when to use each tool
- Add usage instructions or constraints in descriptions
- Configure input validation schemas

Example customizations in this demo:
- Clear, specific descriptions for each tool
- Guidance on when to call related tools (e.g., "Always call get_features-id after this tool...")
- Proper input schemas with required parameters
- Only exposing read endpoints initially (GET operations) for safety

### 4. Start the MCP Server

First, we need to create a configmap to contain the mcpfile.yaml:

```bash
kubectl create cm automcp-config --from-file=mcpfile.yaml
```

Next, we deploy the AutoMCP server:

```bash
cd openshift
kubectl apply -f config/deployment.yaml -f config/service.yaml -f config/route.yaml
```

The MCP service will now be exposed through the `automcp-demo` route, at path `/mcp`. To connect to the server, you will need to use the `streamablehttp` protocol
and the url `<automcp demo route url>/mcp`.

## Key AutoMCP HTTP Conversion Features

- **Automatic Tool Generation**: HTTP endpoints become MCP tools automatically from OpenAPI specs
- **Path Parameter Substitution**: URL templates like `{id}` are handled automatically  
- **Schema Validation**: Input parameters are validated before API calls
- **Streamable HTTP Protocol**: Real-time communication via `streamablehttp`
- **Flexible Configuration**: Full control over which endpoints to expose and how
- **POST/PUT/DELETE Support**: Can expose write operations like adding features, voting, completing, and deleting
