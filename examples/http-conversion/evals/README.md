# Evaluating MCP Servers with gevals

This directory contains evaluation tests for our MCP server using [gevals](https://github.com/genmcp/gevals) - a tool that tests MCP servers by having AI agents complete real tasks.

## What is gevals?

gevals validates your MCP server by having an AI agent attempt to complete tasks using only the tools exposed by your server. It records all tool calls, analyzes the agent's behavior, and checks whether the tools are discoverable, clear, and correctly implemented.

The workflow looks like this:
```
AI Agent → MCP Proxy (recording) → MCP Server
```

gevals records:
- Which tools were called
- What arguments were passed
- When calls were made
- Server responses

## Files in this directory

- **`eval.yaml`** - Main evaluation configuration that ties everything together
- **`claude-code.yaml`** - Agent configuration (defines how to run the Claude Code agent)
- **`mcp-config.yaml`** - MCP server connection details
- **`mcpfile.yaml`** - The MCP tool definitions being evaluated
- **`mcpserver.yaml`** - The MCP server configuration (runtime settings)
- **`tasks/`** - Directory containing task definitions and expected outcomes

## Running the evaluation

1. Clone and build gevals (in the gevals repository directory):
   ```bash
   git clone https://github.com/genmcp/gevals.git
   cd gevals
   go build -o gevals ./cmd/gevals
   ```

2. Start your MCP server using genmcp (if not already running):
   ```bash
   genmcp run -f mcpfile.yaml -s mcpserver.yaml
   ```
   This will start the server at localhost:8080

3. Run the evaluation (from the gevals directory, pointing to this evals folder):
   ```bash
   ./gevals eval /path/to/this/evals/eval.yaml
   ```
   The command output will show you what succeeded or failed.

4. For detailed information, review the output file: `gevals-startup-issue-tracker-evals-out.json`

## Example: The "most-requested-feature" Task

Our evaluation includes a task that asks: **"What is the most requested feature for my app?"**

The task expects the agent to:
1. Call `get_features-top` to get the top feature
2. Call `get_features-id` to get full details about that feature
3. Respond with information about dark mode being the most requested feature

### Initial Failure & The Fix

When we first ran this evaluation, it **failed** because the agent only called `get_features-top` but didn't call `get_features-id` to get the full details. This happened because the tool description didn't make it clear that both calls were necessary.

**The problem:** The original `get_features-top` description was:
```yaml
description: Returns the feature with the most upvotes
```

**The solution:** We updated the description in `mcpfile.yaml` (line 43) to:
```yaml
description: Returns the feature with the most upvotes. Always call get_features-id to give the user all the details about the top requested feature
```

This demonstrates the core value of gevals: **it helps you discover and fix discoverability issues in your MCP server by showing how AI agents actually interact with your tools.**

## Understanding the Assertions

In `eval.yaml`, we define assertions that must pass for the evaluation to succeed:

```yaml
assertions:
  toolsUsed:
    - server: features
      tool: get_features-top
    - server: features
      tool: get_features-id
```

This asserts that both tools must be called during the task. If the agent doesn't discover or use these tools, the evaluation fails, signaling that your tool descriptions or server design needs improvement.

## Key Takeaways

✅ **Use gevals to validate your MCP server design** - Not just functionality, but discoverability

✅ **Tool descriptions matter** - They guide the AI agent to use tools correctly

✅ **Assertions help enforce best practices** - Ensure agents use tools in the intended sequence

✅ **Iterative improvement** - Use failed evaluations to refine your tool descriptions and schemas

## Next Steps

1. Add more tasks to the `tasks/` directory to test other tool combinations
2. Run evaluations regularly as you add new tools or modify existing ones
3. Use LLM judge verification (configured in `eval.yaml`) to validate response quality
4. Review `gevals-*-out.json` files to understand agent behavior patterns

For more information, visit the [gevals repository](https://github.com/genmcp/gevals).
