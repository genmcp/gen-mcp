---
layout: page
title: Ollama Integration Example
description: Learn how to connect local language models to AI assistants using gen-mcp with Ollama
---

# Ollama Integration Example

**[📹 Watch the demo video](https://youtu.be/yqJV9rNwfg8)** to see this example in action!

> **Note:** This video was recorded before the project was renamed from `automcp` to `gen-mcp`. The functionality remains the same—just replace `automcp` with `genmcp` in commands.

## Overview

This example demonstrates how to expose [Ollama](https://ollama.com/), a popular local language model runtime, as MCP tools using gen-mcp. By wrapping Ollama's functionality, you can enable AI assistants to interact with your local language models seamlessly—no custom server code required.

### What You'll Learn

- How to expose HTTP APIs as MCP tools
- How to wrap CLI commands as MCP tools
- The difference between HTTP-based and CLI-based integrations
- Core gen-mcp configuration concepts for tool definitions

### Prerequisites

- **Ollama installed**: Download from [ollama.com](https://ollama.com)
- **gen-mcp installed**: See the [Quick Start guide]({{ '/' | relative_url }}#quick-start)
- **Basic understanding of YAML**: For configuration files

## Two Integration Approaches

gen-mcp supports two different methods for integrating with Ollama, each with its own advantages:

### HTTP-Based Integration (Recommended)

The HTTP approach calls Ollama's REST API directly:

**Advantages:**
- ✅ More reliable with structured JSON responses
- ✅ Better error handling
- ✅ Supports advanced features like streaming control
- ✅ Easier to debug

**Use when:** You want production-grade integration with complete feature access

### CLI-Based Integration

The CLI approach executes `ollama` commands directly:

**Advantages:**
- ✅ Simpler configuration
- ✅ Works without HTTP endpoint
- ✅ Familiar to command-line users

**Use when:** You need quick prototyping or prefer command-line interaction

## HTTP-Based Integration Tutorial

Let's walk through creating an HTTP-based Ollama integration step-by-step.

### Step 1: Start Ollama

Ensure Ollama is running locally:

```bash
ollama serve
```

Ollama will start on `http://localhost:11434` by default. You can verify it's running:

```bash
curl http://localhost:11434
# Should return: "Ollama is running"
```

### Step 2: Understanding the Configuration

GenMCP uses two separate files. Here's the complete configuration:

**Tool Definitions File** (`ollama-http-mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: ollama
version: "1.0.0"
tools:
- name: generate
  title: "Generate a response"
  description: "Generates a response for a given prompt."
  inputSchema:
    type: object
    properties:
      model:
        type: string
        description: "The name of the model to use."
      prompt:
        type: string
        description: "The prompt to generate a response for."
      system:
        type: string
        description: "A system message to override the model's default behavior."
      stream:
        type: boolean
        description: "Whether to stream the response. Must be false."
    required:
    - model
    - prompt
    - stream
  invocation:
    http:
      method: POST
      url: http://localhost:11434/api/generate

- name: chat
  title: "Generate a chat response"
  description: "Generates a response for a chat-based conversation."
  inputSchema:
    type: object
    properties:
      model:
        type: string
        description: "The name of the model to use."
      messages:
        type: array
        items:
          type: object
          properties:
            role:
              type: string
              description: "The role: 'user' or 'assistant'."
            content:
              type: string
              description: "The message content."
          required:
          - role
          - content
      stream:
        type: boolean
        description: "Whether to stream. Must be false."
    required:
    - model
    - messages
    - stream
  invocation:
    http:
      method: POST
      url: http://localhost:11434/api/chat

- name: tags
  title: "List downloaded models"
  description: "Lists all downloaded models."
  inputSchema:
    type: object
    properties: {}
  invocation:
    http:
      method: GET
      url: http://localhost:11434/api/tags

- name: pull_model
  title: "Pull model"
  description: "Download a model from the ollama library."
  inputSchema:
    type: object
    properties:
      model:
        type: string
        description: "The name of the model to pull."
      stream:
        type: boolean
        description: "Must be false."
    required:
    - model
    - stream
  invocation:
    http:
      method: POST
      url: http://localhost:11434/api/pull

- name: running_models
  title: "Get running models"
  description: "List models currently loaded into memory."
  inputSchema:
    type: object
  invocation:
    http:
      method: GET
      url: http://localhost:11434/api/ps
```

**Server Config File** (`ollama-http-mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8009
```

### Step 3: Configuration Breakdown

Let's understand each section:

#### Runtime Configuration (Server Config File)

```yaml
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 8009
```

- `transportProtocol: streamablehttp`: Uses HTTP streaming protocol for real-time communication
- `port: 8009`: The MCP server will listen on this port

#### Tool Definition Structure (Tool Definitions File)

Each tool follows this pattern:

```yaml
- name: generate              # Unique tool identifier
  title: "Generate a response" # Human-readable name
  description: "..."           # What the tool does (for LLM understanding)
  inputSchema:                 # JSON Schema for input validation
    type: object
    properties:
      model:
        type: string
        description: "..."
    required:
    - model
  invocation:                  # How to execute the tool
    http:
      method: POST
      url: http://localhost:11434/api/generate
```

**Key concepts:**

- **name**: Must be unique across all tools
- **description**: Help LLMs understand when to use this tool
- **inputSchema**: Validates inputs before calling Ollama
- **invocation**: Maps the tool to an HTTP endpoint

### Step 4: Run the MCP Server

Start the gen-mcp server with both configuration files:

```bash
genmcp run -f ollama-http-mcpfile.yaml -s ollama-http-mcpserver.yaml
```

You should see:

```
INFO    runtime/server.go:138	Setting up streamable HTTP server	{"port": 8009, "base_path": "/mcp", "stateless": true}
INFO    runtime/server.go:181	Starting MCP server on port 8009
INFO    runtime/server.go:196	Starting HTTP server
```

### Step 5: Test Your Integration

You can now connect an MCP client (like Claude Desktop or any MCP-compatible tool) to `http://localhost:8009/mcp` and use the Ollama tools.

Example tool calls:

**List available models:**
```json
{
  "tool": "tags"
}
```

**Generate a completion:**
```json
{
  "tool": "generate",
  "arguments": {
    "model": "llama2",
    "prompt": "Explain quantum computing in simple terms",
    "stream": false
  }
}
```

## CLI-Based Integration Tutorial

The CLI approach is simpler but more limited. Here's the complete configuration:

### CLI Configuration Files

**Tool Definitions File** (`ollama-cli-mcpfile.yaml`):

```yaml
kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: Ollama
version: 0.0.1
tools:
- name: start_ollama
  title: Start Ollama
  description: Start ollama. Only run if not already started.
  inputSchema:
    type: object
  invocation:
    cli:
      command: nohup ollama start > /dev/null 2>&1 &

- name: check_ollama_running
  title: Check if Ollama is Running
  description: Check if Ollama is running.
  inputSchema:
    type: object
  invocation:
    cli:
      command: curl http://localhost:11434 || echo "ollama is not running"

- name: pull_model
  title: Pull model
  description: Pull a model so that Ollama can use it
  inputSchema:
    type: object
    properties:
      model:
        type: string
        description: The name of the model to pull
  invocation:
    cli:
      command: ollama pull {model}

- name: list_models
  title: List models
  description: List all models ollama has pulled currently.
  inputSchema:
    type: object
  invocation:
    cli:
      command: ollama list

- name: generate_completion
  title: Generate completion
  description: Generate a completion from Ollama
  inputSchema:
    type: object
    properties:
      model:
        type: string
        description: The name of the model to use
      prompt:
        type: string
        description: The prompt to generate a response for
    required:
      - model
      - prompt
  invocation:
    cli:
      command: ollama run {model} {prompt}
```

**Server Config File** (`ollama-cli-mcpserver.yaml`):

```yaml
kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 7008
```

### Running the CLI-Based Server

```bash
genmcp run -f ollama-cli-mcpfile.yaml -s ollama-cli-mcpserver.yaml
```

## Summary

Both HTTP and CLI integrations require two files:
- **Tool Definitions File**: Defines the tools (what capabilities are available)
- **Server Config File**: Defines runtime configuration (how the server runs)

For HTTP-based integrations, tools call Ollama's REST API. For CLI-based integrations, tools execute shell commands directly.

## Next Steps

- **Read the GenMCP Config File Format Guide**: Deep dive into [tool definitions]({{ '/mcpfile.html' | relative_url }}) and [server configuration]({{ '/mcpserver.html' | relative_url }})

## Understanding Input Schema Validation

Input schemas ensure tools receive valid data before execution. Here's how they work:

### Basic Schema

```yaml
inputSchema:
  type: object
  properties:
    model:
      type: string
      description: "Model name"
  required:
  - model
```

**Validation behavior:**
- ✅ `{"model": "llama2"}` - Valid
- ❌ `{}` - Missing required field
- ❌ `{"model": 123}` - Wrong type

### Array Schema

```yaml
inputSchema:
  type: object
  properties:
    messages:
      type: array
      items:
        type: object
        properties:
          role:
            type: string
          content:
            type: string
        required:
        - role
        - content
```

This validates complex nested structures like chat messages.

## Common Patterns and Best Practices

### Pattern 1: Tool Chaining

Design tools to work together:

```yaml
- name: check_ollama_running
  description: "Check if Ollama is running. Run before other tools."

- name: pull_model
  description: "Download a model. Run check_ollama_running first."
```

### Pattern 2: Safe Defaults

Require explicit flags for dangerous operations:

```yaml
inputSchema:
  properties:
    stream:
      type: boolean
      description: "Must be false for MCP compatibility"
  required:
  - stream
```

### Pattern 3: Clear Descriptions

Help LLMs understand tool usage:

```yaml
description: "Download a model from the ollama library. This may take several minutes for large models. Always check if the model exists first using tags."
```

## Troubleshooting

### Issue: "Connection refused" error

**Solution:** Ensure Ollama is running:
```bash
ollama serve
```

### Issue: "Model not found" error

**Solution:** Pull the model first:
```bash
ollama pull llama2
```

### Issue: Tools not appearing in MCP client

**Solution:** Check the server is running and the port matches your client configuration.

## Next Steps

- **Explore HTTP Conversion**: Learn how to convert any REST API to MCP tools in the [HTTP Conversion Example]({{ '/example-http-conversion.html' | relative_url }})
- **Read the GenMCP Config File Format Guide**: Deep dive into [tool definitions]({{ '/mcpfile.html' | relative_url }}) and [server configuration]({{ '/mcpserver.html' | relative_url }})
- **Join the Community**: Get help on [Discord](https://discord.gg/AwP6GAUEQR)

## Resources

- [Ollama Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Example Files on GitHub](https://github.com/genmcp/gen-mcp/tree/main/examples/ollama)
