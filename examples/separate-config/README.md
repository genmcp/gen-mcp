# Separate Configuration Example

This example demonstrates the configuration structure where runtime settings are separated from tool definitions.

## Files

- `mcpserver.yaml` - Server runtime configuration (port, transport protocol, etc.) - **REQUIRED**
- `mcpfile.yaml` - Tool, prompt, and resource definitions

## Running the server

```bash
gen-mcp run -f mcpfile.yaml -s mcpserver.yaml
```

## Benefits of Separation

1. **Clear separation of concerns**: Runtime configuration is separate from capability definitions
2. **Easier configuration management**: Different environments can use different server configs with the same tool definitions
3. **Future extensibility**: Server config can be extended with logging, tracing, and other operational settings without cluttering the tool definitions
