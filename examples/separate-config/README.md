# Separate Configuration Example

This example demonstrates the new configuration structure where runtime settings are separated from tool definitions.

## Files

- `mcpserver.yaml` - Server runtime configuration (port, transport protocol, etc.)
- `mcpfile.yaml` - Tool, prompt, and resource definitions

## Running the server

```bash
gen-mcp run -f mcpfile.yaml -s mcpserver.yaml
```

## Benefits of Separation

1. **Clear separation of concerns**: Runtime configuration is separate from capability definitions
2. **Easier configuration management**: Different environments can use different server configs with the same tool definitions
3. **Future extensibility**: Server config can be extended with logging, tracing, and other operational settings without cluttering the tool definitions
4. **Backward compatible**: You can still use a single `mcpfile.yaml` with runtime included

## Backward Compatibility

If you prefer the traditional approach, you can still include the runtime configuration in `mcpfile.yaml`:

```bash
gen-mcp run -f mcpfile.yaml
```
