# 🤖 Talk to Ollama with AutoMCP!

This example demonstrates how to use AutoMCP to wrap the Ollama API, instantly turning your local LLMs into tools accessible via the MCP standard.

It's like giving your AI a superpower! 🚀

## How to Use It

1.  **Make sure Ollama is running** locally (usually at `http://localhost:11434`).
2.  With `automcp` in your path, run the `mcpfile` with `automcp`:

    ```bash
    automcp run -f examples/ollama/mcpfile.yaml
    ```

And that's it! AutoMCP will start a server, exposing Ollama endpoints as tools, allowing you to generate completions, list running models, pull new models, and more. You can now send requests to this server to interact with your Ollama models as if they were any other MCP tool.
