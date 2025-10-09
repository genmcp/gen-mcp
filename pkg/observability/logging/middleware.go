package logging

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

type ctxKey struct{}

// WithLoggingMiddleware creates an MCP middleware that adds request-scoped logging.
// It extracts the ServerSession from incoming requests and creates a request-specific
// logger that can be retrieved using FromContext. If session extraction or logger
// creation fails, it logs a warning and continues the request chain without error.
func WithLoggingMiddleware(base *zap.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			session := req.GetSession()
			ss, ok := session.(*mcp.ServerSession)
			if !ok {
				// don't error out - just continue the request chain
				base.Warn("session on request was not ServerSession, not adding logger")
				return next(ctx, method, req)
			}

			requestLogger, err := NewRequestLogger(ctx, base, ss)
			if err != nil {
				base.Warn("failed to initialize request logger", zap.Error(err))
				return next(ctx, method, req)
			}

			ctx = WithRequestLogger(ctx, requestLogger)
			return next(ctx, method, req)
		}
	}
}

// WithRequestLogger stores a logger in the given context, making it available
// for retrieval via FromContext throughout the request lifecycle.
func WithRequestLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext retrieves the logger stored in the context by WithRequestLogger.
// If no logger is found or the stored value is not a *zap.Logger, it returns
// a no-op logger to ensure safe operation without panics.
func FromContext(ctx context.Context) *zap.Logger {
	logger := ctx.Value(ctxKey{})
	if logger == nil {
		return zap.NewNop()
	}

	zapLogger, ok := logger.(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return zapLogger
}
