---
layout: page
title: HTTP API Conversion Example
description: Learn how to automatically convert REST APIs into MCP tools using gen-mcp's OpenAPI conversion
---

# HTTP API Conversion Example

**[📹 Watch the demo video](https://youtu.be/boMyFzpgJoA)** to see this example in action!

> **Note:** This video was recorded before the project was renamed from `automcp` to `gen-mcp`. The functionality remains the same—just replace `automcp` with `genmcp` in commands.

## Overview

This example demonstrates gen-mcp's most powerful feature: **automatic conversion of HTTP REST APIs into MCP tools**. Instead of writing custom MCP server code, you simply point gen-mcp at an OpenAPI specification, and it instantly creates a working MCP server with all endpoints exposed as callable tools.

### What You'll Learn

- How to convert OpenAPI specs to MCP configurations
- Path parameter substitution in URLs (`/features/{id}`)
- Input schema validation for API calls
- Customizing generated configurations
- Using invocation bases for cleaner configs
- Best practices for API-to-MCP conversion

### Prerequisites

- **gen-mcp installed**: See the [Quick Start guide]({{ '/' | relative_url }}#quick-start)
- **Go installed** (for running the example API): Download from [go.dev](https://go.dev)
- **Basic understanding of REST APIs**: HTTP methods, JSON, etc.

## The Example API

This tutorial uses a simple **Feature Request API** that manages product feature requests. It demonstrates common REST API patterns:

- **GET** endpoints for retrieving data
- **POST** endpoints for creating and updating data
- **DELETE** endpoints for removing data
- Path parameters for resource identification
- JSON request/response bodies

### API Endpoints

| Method | Endpoint             | Description                   |
|--------|----------------------|-------------------------------|
| GET    | `/features`          | List all features (summaries) |
| GET    | `/features/top`      | Get most-voted feature        |
| GET    | `/features/{id}`     | Get detailed feature by ID    |
| POST   | `/features`          | Create new feature request    |
| POST   | `/features/vote`     | Vote for a feature            |
| POST   | `/features/complete` | Mark feature as completed     |
| DELETE | `/features/{id}`     | Delete a feature              |
| GET    | `/openapi.json`      | OpenAPI specification         |

## Step-by-Step Tutorial

### Step 1: Clone the gen-mcp Repository

First, clone the gen-mcp repository to get the example code:

```bash
git clone https://github.com/genmcp/gen-mcp.git
cd gen-mcp
```

### Step 2: Start the Example API Server

Navigate to the example and start the Feature Request API server:

```bash
cd examples/http-conversion/feature-requests
go run main.go
```

You should see:

```
Feature request server starting on :9090
Endpoints:
  GET    /features/top          - Get most voted feature (summary)
  GET    /features/{id}         - Get feature details
  POST   /features              - Add new feature
  ...
```

The API is now running at `http://localhost:9090`.

### Step 3: Test the API

Let's verify the API works before converting it:

```bash
# List all features
curl http://localhost:9090/features

# Get a specific feature
curl http://localhost:9090/features/1

# Get the OpenAPI specification
curl http://localhost:9090/openapi.json
```

You should see JSON responses with feature data.

### Step 4: Convert the API to MCP

Now for the magic! Use gen-mcp's `convert` command to automatically generate an MCP configuration:

```bash
# Navigate back to the http-conversion directory
cd ..

# Convert the OpenAPI spec to mcpfile.yaml
genmcp convert http://localhost:9090/openapi.json
```

gen-mcp will analyze the OpenAPI specification and create two files: a tool definitions file (`mcpfile.yaml`) and a server config file (`mcpserver.yaml`).

You should see output like:

```
INFO    Fetching OpenAPI spec from http://localhost:9090/openapi.json
INFO    Converted 7 endpoints to MCP tools
INFO    Created mcpfile.yaml
INFO    Created mcpserver.yaml
```

### Step 5: Understanding the Generated Configuration

gen-mcp creates two separate files:

1. **Tool Definitions File** (`mcpfile.yaml`) - Contains all the tools, prompts, resources, and invocation bases
2. **Server Config File** (`mcpserver.yaml`) - Contains the server runtime configuration

Let's examine the generated files. Note that the tool definitions file includes all 7 tools from the API—we'll show a few key examples here to understand the structure:

**Tool Definitions File** (`mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: Feature Request API
version: 0.0.1

invocationBases:
  baseApi:
    http:
      url: http://localhost:9090

# The generated file includes all 7 tools - showing key examples here
tools:
- name: get_features
  title: Get all features
  description: Returns a list of all features sorted by upvotes
  inputSchema:
    type: object
  invocation:
    extends:
      from: baseApi
      extend:
        url: /features
      override:
        method: GET

- name: get_features-id
  title: Get feature details
  description: Returns detailed information about a specific feature
  inputSchema:
    type: object
    properties:
      id:
        type: integer
    required:
    - id
  invocation:
    extends:
      from: baseApi
      extend:
        url: /features/{id}
      override:
        method: GET

- name: post_features
  title: Add new feature
  description: Create a new feature request
  inputSchema:
    type: object
    properties:
      title:
        type: string
        description: Feature title
      description:
        type: string
        description: Detailed description
      details:
        type: string
        description: Implementation notes
    required:
    - title
  invocation:
    extends:
      from: baseApi
      extend:
        url: /features
      override:
        method: POST

- name: post_features-vote
  title: Vote for feature
  description: Increment the upvote count for a specific feature
  inputSchema:
    type: object
    properties:
      feature_id:
        type: integer
        description: ID of the feature to vote for
    required:
    - feature_id
  invocation:
    extends:
      from: baseApi
      extend:
        url: /features/vote
      override:
        method: POST
```

### Step 6: Configuration Deep Dive

Let's understand the key concepts:

#### Invocation Bases

```yaml
invocationBases:
  baseApi:
    http:
      url: http://localhost:9090
```

**Purpose**: Define reusable HTTP configuration
- Avoids repeating `http://localhost:9090` in every tool
- Easy to change the base URL in one place
- Cleaner, more maintainable configuration

#### Tool with Path Parameters

```yaml
- name: get_features-id
  inputSchema:
    properties:
      id:
        type: integer
    required:
    - id
  invocation:
    extends:
      from: baseApi
      extend:
        url: /features/{id}
```

**How it works:**
1. Tool receives `{"id": 1}` as input
2. gen-mcp validates `id` is an integer
3. gen-mcp substitutes `{id}` in URL with `1`
4. Final URL: `http://localhost:9090/features/1`

#### Extends Invocation

```yaml
invocation:
  extends:
    from: baseApi          # Reference base configuration
    extend:
      url: /features       # Append to base URL
    override:
      method: GET          # Set HTTP method
```

**Operations:**
- `from`: Which base to extend
- `extend`: Merge/append values (URL concatenation)
- `override`: Replace values completely (HTTP method)

### Step 7: Customize the Configuration

The generated configuration is a starting point. Let's improve it:

#### Better Descriptions

Change:
```yaml
description: Returns detailed information about a specific feature
```

To:
```yaml
description: |
  Returns complete details for a specific feature including title, description,
  implementation details, upvote count, and completion status. Use this after
  calling get_features to get full information about a feature of interest.
```

**Why?** LLMs use descriptions to decide when to call tools. Better descriptions lead to better tool usage.

#### Add Safety Guards

For destructive operations:

```yaml
- name: delete_features-id
  title: Delete feature
  description: |
    ⚠️ DESTRUCTIVE: Permanently deletes a feature request. This cannot be undone.
    Always confirm with the user before calling this tool. Consider using
    post_features-complete instead to mark features as done without deleting.
```

#### Add Tool Instructions

```yaml
- name: get_features
  description: |
    Returns summaries of all features sorted by upvotes (highest first).
    Use this to get an overview. For detailed information about a specific
    feature, call get_features-id with the feature's ID.
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

### Step 8: Run the MCP Server

Start your customized MCP server with both files:

```bash
genmcp run -f mcpfile.yaml -s mcpserver.yaml
```

You should see:

```
...
INFO    runtime/server.go:138   Setting up streamable HTTP server   {"port": 8080, "base_path": "/mcp", "stateless": true}
INFO    runtime/server.go:181   Starting MCP server on port 8080
INFO    runtime/server.go:196   Starting HTTP server
```

### Step 9: Connect an MCP Client

Now you can connect any MCP-compatible client (like Claude Desktop) to `http://localhost:8080/mcp`.

The client will discover all your tools and can call them:

```json
// Example: List all features
{
  "tool": "get_features"
}

// Example: Get specific feature
{
  "tool": "get_features-id",
  "arguments": {
    "id": 1
  }
}

// Example: Create new feature
{
  "tool": "post_features",
  "arguments": {
    "title": "Export to PDF",
    "description": "Allow users to export data as PDF",
    "details": "Support for custom templates and styling"
  }
}
```

## Advanced Concepts

### Input Schema Validation

gen-mcp validates all inputs before calling the API:

```yaml
inputSchema:
  type: object
  properties:
    feature_id:
      type: integer
      description: ID of the feature to vote for
  required:
  - feature_id
```

**What happens:**
- ✅ `{"feature_id": 1}` → Valid, calls API
- ❌ `{"feature_id": "one"}` → Error: type mismatch
- ❌ `{}` → Error: missing required field

This prevents invalid API calls and provides clear error messages.

### Complex Input Schemas

For nested data structures:

```yaml
inputSchema:
  type: object
  properties:
    title:
      type: string
      minLength: 3
      maxLength: 100
    description:
      type: string
    tags:
      type: array
      items:
        type: string
      maxItems: 5
```

**Validation features:**
- Type checking (string, integer, boolean, array, object)
- String length constraints
- Array size limits
- Nested object validation
- Required vs. optional fields

### POST Request Bodies

For POST requests, gen-mcp automatically:
1. Validates the input schema
2. Converts inputs to JSON
3. Sets `Content-Type: application/json`
4. Sends JSON in request body

Example tool call:
```json
{
  "tool": "post_features",
  "arguments": {
    "title": "My Feature",
    "description": "Description here"
  }
}
```

Becomes HTTP request:
```http
POST /features HTTP/1.1
Host: localhost:9090
Content-Type: application/json

{
  "title": "My Feature",
  "description": "Description here"
}
```

### Multiple Path Parameters

URLs can have multiple parameters:

```yaml
invocation:
  extends:
    from: baseApi
    extend:
      url: /users/{userId}/posts/{postId}
```

gen-mcp substitutes all parameters:
- Input: `{"userId": 42, "postId": 123}`
- URL: `http://localhost:9090/users/42/posts/123`

## Common Patterns

### Pattern 1: Read-Only Tools First

Start by exposing only GET endpoints:

```yaml
tools:
- name: get_features
  invocation:
    http:
      method: GET
      url: http://localhost:9090/features
```

**Why?** Read-only operations are safer for initial testing.

### Pattern 2: Tool Chaining

Design tools that work together:

```yaml
- name: get_features
  description: "Lists all features. Use get_features-id for details on any feature."

- name: get_features-id
  description: "Gets full details for a feature. Get the ID from get_features first."
```

### Pattern 3: Descriptive Naming

Use clear, action-oriented names:

- ✅ `get_feature_details`
- ✅ `create_feature_request`
- ✅ `vote_for_feature`
- ❌ `feature1`
- ❌ `endpoint_2`
- ❌ `api_call`

## Real-World Applications

### Use Case 1: Internal APIs

Expose your company's internal APIs to AI assistants:

```yaml
tools:
- name: get_customer_by_email
  description: "Look up customer records by email address"
  invocation:
    http:
      method: GET
      url: https://api.internal.company.com/customers/{email}
```

### Use Case 2: SaaS Platform Integration

Connect third-party services:

```yaml
invocationBases:
  stripeApi:
    http:
      url: https://api.stripe.com/v1
      headers:
        Authorization: "Bearer {env.STRIPE_API_KEY}"

tools:
- name: list_customers
  invocation:
    extends:
      from: stripeApi
      extend:
        url: /customers
```

### Use Case 3: Microservices

Expose multiple microservices as one MCP server:

```yaml
invocationBases:
  authService:
    http:
      url: http://auth-service:8080

  dataService:
    http:
      url: http://data-service:8080

tools:
- name: authenticate_user
  invocation:
    extends:
      from: authService
      extend:
        url: /auth/login

- name: get_user_data
  invocation:
    extends:
      from: dataService
      extend:
        url: /data/users/{userId}
```

## Testing Your MCP Server with gevals

Once you have your MCP server running, how do you know if it's working well? That's where **[gevals](https://github.com/genmcp/gevals)** comes in—a testing framework that validates your MCP server by having AI agents complete real tasks using your tools.

### What is gevals?

gevals tests your MCP server by:
1. **Running an AI agent** that attempts to complete tasks using your tools
2. **Recording all interactions** - which tools were called, with what arguments, and when
3. **Verifying outcomes** - checking if the agent successfully completed the task
4. **Identifying issues** - discovering problems with tool discoverability, descriptions, or implementation

**The workflow:**
```
AI Agent → MCP Proxy (recording) → Your MCP Server
```

### Why Use gevals?

Testing MCP servers is different from testing regular APIs. You need to know:
- Can AI agents **discover** the right tools for a task?
- Are tool **descriptions** clear enough to guide usage?
- Do tools work together in the **expected sequence**?
- Are **error messages** helpful for the agent?

gevals answers these questions by simulating real AI agent behavior.

### Example: Testing the Feature Request API

This example includes a gevals test suite in the `evals/` directory. Let's look at a real test case:

**Task:** "What is the most requested feature for my app?"

**Expected behavior:**
1. Agent calls `get_features-top` to find the top feature
2. Agent calls `get_features-id` to get full details
3. Agent responds with information about the top feature

### A Real Bug Discovery

When this test first ran, it **failed**. The agent only called `get_features-top` but never called `get_features-id` for the full details.

**The problem:** The tool description wasn't clear enough:
```yaml
description: Returns the feature with the most upvotes
```

**The fix:** Updated description to guide the agent:
```yaml
description: Returns the feature with the most upvotes. Always call get_features-id to give the user all the details about the top requested feature
```

After this change, the test passed! This demonstrates how gevals helps you **improve tool discoverability** through iterative testing.

### Running the Evaluation

To run the gevals test suite for this example:

1. **Install gevals:**
   ```bash
   git clone https://github.com/genmcp/gevals.git
   cd gevals
   go build -o gevals ./cmd/gevals
   ```

2. **Start the MCP server** (in the gen-mcp repo):
   ```bash
   cd examples/http-conversion
   genmcp run -f mcpfile.yaml
   ```

3. **Run the evaluation** (from the gevals directory):
   ```bash
   ./gevals eval /path/to/gen-mcp/examples/http-conversion/evals/eval.yaml
   ```

4. **Review results:**
   - Console output shows pass/fail status
   - `gevals-*-out.json` contains detailed interaction logs

### Understanding Assertions

The evaluation configuration defines what must happen:

```yaml
assertions:
  toolsUsed:
    - server: features
      tool: get_features-top
    - server: features
      tool: get_features-id
```

If the agent doesn't use both tools, the test fails—signaling a discoverability problem.

### Key Takeaways

✅ **Test discoverability, not just functionality** - Tools must be findable and understandable by AI agents

✅ **Tool descriptions are critical** - They're the primary way agents decide when to use a tool

✅ **Iterate based on failures** - Failed tests reveal how to improve your MCP server design

✅ **Verify tool sequencing** - Ensure agents use tools in the intended order

### Learn More

- **[gevals GitHub Repository](https://github.com/genmcp/gevals)** - Full documentation and examples
- **[Evaluation Files](https://github.com/genmcp/gen-mcp/tree/main/examples/http-conversion/evals)** - Complete test suite for this example

## Troubleshooting

### Issue: "Failed to fetch OpenAPI spec"

**Solution:** Ensure the API server is running and the URL is correct:
```bash
curl http://localhost:9090/openapi.json
```

### Issue: "Invalid URL template"

**Solution:** Check that path parameters match input schema properties:
```yaml
# URL has {id}
url: /features/{id}

# Input schema must have 'id' property
inputSchema:
  properties:
    id:
      type: integer
```

### Issue: "Tool not found"

**Solution:** Verify the MCP server loaded the tool:
```bash
# Check server logs for:
INFO    Loaded 7 tools: get_features, get_features-id, ...
```

## Next Steps

- **Explore Ollama Integration**: Learn CLI-based tool wrapping in the [Ollama Example]({{ '/example-ollama.html' | relative_url }})
- **Read the MCP File Format Guide**: Master advanced configuration in the [tool definitions guide]({{ '/mcpfile.html' | relative_url }}) and [server config guide]({{ '/mcpserver.html' | relative_url }})
- **Join the Community**: Share your API conversions on [Discord](https://discord.gg/AwP6GAUEQR)

## Resources

- [OpenAPI Specification](https://swagger.io/specification/)
- [MCP Protocol Documentation](https://modelcontextprotocol.io/)
- [Example Files on GitHub](https://github.com/genmcp/gen-mcp/tree/main/examples/http-conversion)
- [gen-mcp Convert Command Reference]({{ '/' | relative_url }}#commands)
