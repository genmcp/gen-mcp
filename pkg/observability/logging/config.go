// Package logging provides logging integration between zap loggers and
// Model Context Protocol (MCP) clients.
//
// This package enables applications to send logs both to traditional outputs
// (files, console) via zap and to MCP clients via ServerSession.
//
// # Basic Usage
//
// At application startup, create a base logger:
//
//	cfg := &logging.LoggingConfig{
//		Config: zap.Config{
//			Level:    zap.NewAtomicLevelAt(zap.InfoLevel),
//			Encoding: "json",
//			OutputPaths: []string{"stdout"},
//		},
//	}
//	baseLogger, err := cfg.BuildBase()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer baseLogger.Sync()
//
// Use the base logger for general application logging:
//
//	baseLogger.Info("Application starting", zap.String("version", "1.0"))
//
// In request handlers where you have access to mcp.ServerSession,
// create a request-scoped logger that sends to both destinations:
//
//	func HandleRequest(ctx context.Context, ss *mcp.ServerSession) {
//		reqLogger, err := logging.NewRequestLogger(ctx, baseLogger, ss)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// This log goes to both base logger output AND MCP client
//		reqLogger.Info("Processing request",
//			zap.String("method", "POST"),
//			zap.Duration("elapsed", time.Since(start)),
//		)
//	}
//
// # MCP Integration
//
// The package automatically converts zap log levels to MCP levels:
//   - zap.DebugLevel → "debug"
//   - zap.InfoLevel → "info"
//   - zap.WarnLevel → "warning"
//   - zap.ErrorLevel → "error"
//   - zap.DPanicLevel → "critical"
//   - zap.PanicLevel → "alert"
//   - zap.FatalLevel → "emergency"
//
// All logs are sent to MCP clients regardless of level - the ServerSession
// determines whether to actually transmit them based on client preferences.
package logging

import (
	"fmt"

	"go.uber.org/zap"
)

type LoggingConfig struct {
	zap.Config
	enableMcpLogs *bool
}

// MCPLogsEnabled returns whether the mcp logs are enabled, defaulting to true if unset
func (lc *LoggingConfig) MCPLogsEnabled() bool {
	if lc.enableMcpLogs == nil {
		return true
	}

	return *lc.enableMcpLogs
}

// BuildBase creates a base logger from the configuration.
// This should be called once at application startup and the resulting logger
// should be reused throughout the application's lifetime.
//
// For request-scoped logging that also sends to MCP clients, use NewRequestLogger
// with the base logger returned from this method.
func (lc *LoggingConfig) BuildBase() (*zap.Logger, error) {
	logger, err := lc.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to build base zap logger: %w", err)
	}
	return logger, nil
}
