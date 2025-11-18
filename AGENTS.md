# GenMCP Dev Guidance

GenMCP provides a declarative way to create MCP servers around existing systems
The declarative file format is described in `./docs/mcp_file_format.md`
Some examples can be found in `./examples/`

## Repository Layout

- `./pkg/builder`: builder to create container images from an MCP file that will contain the mcp server
- `./pkg/cli`: cli commands and utilities for the `genmcp` cli
- `./pkg/converter`: packages to automatically convert different types of applications into an MCP file. Currently, there are subpackages for OpenAPI specs and cli conversion with an agent
- `./pkg/invocation/`: core invocation interfaces and utilities. Subpackages for specific invocation types (http, cli)
- `./pkg/mcpfile/`: core types for the mcp file
- `./pkg/mcpserver/`: core mcp server implementation
- `./pkg/oauth/`: OAuth helpers, primarily to be used in `./pkg/mcpserver`
- `./pkg/observability/`: observability packages to be used throughout
- `./cmd/genmcp/`: the entry point for the genmcp CLI
- `./cmd/genmcp-server/`: the entry point for the genmcp server binary (to be used in containers)
- `./examples/`: various examples of how to use genmcp
- `./docs/`: docs around the mcp file spec
- `./specs/`: generated json schema specs for the mcp file format
- `./test/`: integration tests for the project

## Dev commands

There are more makefile commands than those listed here, but these are the only ones you need to worry about while developing genmcp

- `make build`: build everything - in general, this is how to build the project
- `make update-codegen`: this will update the codegen for the json schemas. Call this whenever you make changes to the core types
- `make test`: this will correctly re-build required binaries and run all the tests

## Coding Guidelines

- When adding logs, check whether you are using the server-only logger (embedded in the `MCPServer` type, and accessed from invocation contexts with `logging.BaseFromContext(ctx)`) or the server and client logger. Make sure that any logs with sensitive info use the server-only logger
- When writing unit tests, follow the table test convention used throughout the codebase

## PR Guidelines

- When finishing a set of changes, before opening a PR be sure to update the `./CHANGELOG.md` file with any relevant changes your PR introduces. Every PR with user-visible changes should update the `./CHANGELOG.md` with a short summary of the changes
