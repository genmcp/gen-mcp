# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0]

### Added
- Runtime environment variable overrides in mcpfile (#177)
- Tool annotations support (destructiveHint, idempotentHint, openWorldHint) to indicate tool behavior to clients (#180)
- Server instructions support to provide context to LLMs (#173)
- Comprehensive logging system with invocation and server logs (#168)
- JSON schema validation for mcpfile (#155)
- Support for MCP spec `resource` and `resourceTemplate` primitives (#157)
- Support for MCP spec `Prompts` (#138)
- `genmcp build` command to create container images from mcpfiles (#126)
- AI-based converter for CLI tools (#67)
- Structured output from HTTP JSON responses (#107)
- `genmcp version` command (#105)
- gRPC integration demo showcasing GenMCP with gRPC services (#153)

### Changed
- **BREAKING**: Simplified mcpfile format by embedding server fields directly, migrated format version to v0.1.0 (#137)
- GenMCP now uses the official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk) (#90)
- Bumped MCP Go-SDK to v1.0.0 release (#134)
- StreamableHttp servers are now configurable as stateless or stateful (default: stateless) (#100)
- Migrated from ghodss/yaml to sigs.k8s.io/yaml (#89)

### Deprecated

### Removed
- Vendor directory to reduce PR noise (#154)

### Fixed
- Parsing now returns proper error on invalid mcpfile version (#171)
- OpenAPI 2.0 body parameter handling now correctly aligns with spec (#150)
- Tool input schemas with empty properties now correctly serialize to `{}` (#112)
- OAuth example ports corrected to avoid conflicts (#101)
- Individual tool errors in OpenAPI conversion no longer block entire mcpfile creation (#97)
- Release workflows now target correct branches (#86)
- Nightly release job now manages only a single 'nightly' tag (#83)

### Security

### New Contributors
- @mikelolasagasti made their first contribution
- @Manaswa-S made their first contribution
- @rh-rahulshetty made their first contribution
- @aliok made their first contribution

## [v0.0.0]

### Added
- Initial MCP File specification
- Simple converter to convert OpenAPI v2/v3 specifications into the MCP file format
- Initial MCP Server implementation
  - Reads from the MCP file and runs a server with the provided tools
  - OAuth 2.0/OIDC support for the MCP Client -> MCP Server connection
  - TLS Support for the MCP Client -> MCP Server connection
- Initial genmcp CLI implementation
  - genmcp run will run servers from the MCP files
  - genmcp stop will stop servers
  - genmcp convert converts an OpenAPI spec to an mcp file
- Initial examples
  - CLI/HTTP examples with ollama
  - HTTP conversion examples and integrations with multiple tools
  - Integration with k8s, via ToolHive

### Changed

### Deprecated

### Removed

### Fixed

### Security
