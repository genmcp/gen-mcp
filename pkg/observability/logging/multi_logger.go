package logging

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewRequestLogger creates a request-scoped logger that sends logs to both
// the base logger and the MCP client via ServerSession.
//
// Performance: This function is designed to be called per-request. It reuses
// the pre-built baseLogger (created once at startup) and only creates the
// lightweight MCP core and tee wrapper.
//
// Usage:
//   - Build baseLogger once at startup: baseLogger, _ := cfg.BuildBase()
//   - Call this function per-request: reqLogger, err := NewRequestLogger(ctx, baseLogger, session)
func NewRequestLogger(ctx context.Context, baseLogger *zap.Logger, ss *mcp.ServerSession) (*zap.Logger, error) {
	mcpCore, err := NewMcpCoreWithContext(ctx, ss)
	if err != nil {
		return nil, err
	}

	// AddCallerSkip(1) ensures that caller information points to the actual
	// calling code, not the logger wrapper itself
	return baseLogger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, mcpCore)
		}),
		zap.AddCallerSkip(1),
	), nil
}
