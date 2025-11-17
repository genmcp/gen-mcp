package mcpfile

import (
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

const (
	// TODO: is this duplicated?
	MCPFileVersion         = "0.2.0"
	TransportProtocolStdio = "stdio"
)

var _ invocation.Primitive = (*definitions.Tool)(nil)
var _ invocation.Primitive = (*definitions.Prompt)(nil)
var _ invocation.Primitive = (*definitions.Resource)(nil)
var _ invocation.Primitive = (*definitions.ResourceTemplate)(nil)
