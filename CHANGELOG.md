# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0]

### Added

### Changed
- GenMCP now uses the official [modelcontextprocotol SDK](https://github.com/modelcontextprotocol/go-sdk)
- StreamableHttp servers are now registered as stateless, so you can safely scale them to 0 or to N

### Deprecated

### Removed

### Fixed
- When converting an OpenAPI spec, invalid tools will not cause the conversion to fail anymore

### Security

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
