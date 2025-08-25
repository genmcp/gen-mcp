# 🤖 Talk to Ollama with gen-mcp!

📹 **[Watch the demo video](https://youtu.be/yqJV9rNwfg8)** to see this example in action!

This directory demonstrates two different approaches to integrate Ollama with gen-mcp: HTTP-based and CLI-based methods.

## Two Integration Methods

### HTTP-based Integration (`ollama-http.yaml`)
Uses Ollama's REST API endpoints directly:
- Requires Ollama to be running locally at `http://localhost:11434`
- Provides tools for completions, embeddings, model management via HTTP calls
- More reliable and provides structured JSON responses

### CLI-based Integration (`ollama-cli.yaml`) 
Uses Ollama's command-line interface:
- Executes `ollama` CLI commands directly
- Useful when you prefer command-line interaction
- Provides tools for starting Ollama, pulling models, and generating completions

## How to Use

### HTTP Method (Recommended)
1. **Make sure Ollama is running** locally (usually at `http://localhost:11434`).
2. Run the HTTP-based integration:
   ```bash
   genmcp run -f examples/ollama/ollama-http.yaml
   ```

### CLI Method  
1. **Ensure Ollama is installed** and available in your PATH.
2. Run the CLI-based integration:
   ```bash
   genmcp run -f examples/ollama/ollama-cli.yaml
   ```

Both methods expose Ollama functionality as MCP tools, allowing AI assistants to interact with your local language models seamlessly!
