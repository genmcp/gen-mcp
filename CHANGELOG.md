# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.0]

### Added
- Initial MCP File specification
- Simple converter to convert OpenAPI v2/v3 specifications into the MCP file format
- Initial MCP Server implementation
  - Reads from the MCP file and runs a server with the provided tools
  - OAuth 2.0/OIDC support for the MCP Client -> MCP Server connection
- Initial genmcp CLI implementation
  - genmcp run will run servers from the MCP files
  - genmcp stop will stop servers
  - genmcp convert converts an OpenAPI spec to an mcp file
- Initial examples
  - CLI/HTTP examples with ollama
  - HTTP conversion examples and integrations with multiple tools

### Changed

### Deprecated

### Removed

### Fixed

### Security
