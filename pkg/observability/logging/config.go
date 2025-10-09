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
//	cfg := &logging.LoggingConfig{}
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
	"go.uber.org/zap/zapcore"
)

// LoggingConfig provides a JSON-schema friendly configuration for logging
// that can be converted to a zap.Config when needed.
type LoggingConfig struct {
	// Level is the minimum enabled logging level (debug, info, warn, error, dpanic, panic, fatal)
	Level string `json:"level,omitempty" jsonschema:"optional"`
	// Development puts the logger in development mode
	Development bool `json:"development,omitempty" jsonschema:"optional"`
	// DisableCaller stops annotating logs with the calling function's file name and line number
	DisableCaller bool `json:"disableCaller,omitempty" jsonschema:"optional"`
	// DisableStacktrace completely disables automatic stacktrace capturing
	DisableStacktrace bool `json:"disableStacktrace,omitempty" jsonschema:"optional"`
	// Encoding sets the logger's encoding ("json" or "console")
	Encoding string `json:"encoding,omitempty" jsonschema:"optional"`
	// OutputPaths is a list of URLs or file paths to write logging output to
	OutputPaths []string `json:"outputPaths,omitempty" jsonschema:"optional"`
	// ErrorOutputPaths is a list of URLs to write internal logger errors to
	ErrorOutputPaths []string `json:"errorOutputPaths,omitempty" jsonschema:"optional"`
	// InitialFields is a collection of fields to add to the root logger
	InitialFields map[string]interface{} `json:"initialFields,omitempty" jsonschema:"optional"`
	// EnableMcpLogs controls whether logs are sent to MCP clients
	EnableMcpLogs *bool `json:"enableMcpLogs,omitempty" jsonschema:"optional"`
}

// MCPLogsEnabled returns whether the mcp logs are enabled, defaulting to true if unset
func (lc *LoggingConfig) MCPLogsEnabled() bool {
	if lc.EnableMcpLogs == nil {
		return true
	}

	return *lc.EnableMcpLogs
}

// toZapConfig converts the schema-friendly LoggingConfig to a zap.Config
func (lc *LoggingConfig) toZapConfig() (zap.Config, error) {
	var config zap.Config

	// Set defaults if not specified
	switch lc.Encoding {
	case "console":
		config = zap.NewDevelopmentConfig()
	default:
		config = zap.NewProductionConfig()
	}

	// Override with specified values
	if lc.Level != "" {
		level, err := zapcore.ParseLevel(lc.Level)
		if err != nil {
			return config, fmt.Errorf("invalid log level %q: %w", lc.Level, err)
		}
		config.Level = zap.NewAtomicLevelAt(level)
	}

	if lc.Encoding != "" {
		config.Encoding = lc.Encoding
	}

	config.Development = lc.Development
	config.DisableCaller = lc.DisableCaller
	config.DisableStacktrace = lc.DisableStacktrace

	if len(lc.OutputPaths) > 0 {
		config.OutputPaths = lc.OutputPaths
	}

	if len(lc.ErrorOutputPaths) > 0 {
		config.ErrorOutputPaths = lc.ErrorOutputPaths
	}

	if lc.InitialFields != nil {
		config.InitialFields = lc.InitialFields
	}

	return config, nil
}

// BuildBase creates a base logger from the configuration.
// This should be called once at application startup and the resulting logger
// should be reused throughout the application's lifetime.
//
// For request-scoped logging that also sends to MCP clients, use NewRequestLogger
// with the base logger returned from this method.
func (lc *LoggingConfig) BuildBase() (*zap.Logger, error) {
	config, err := lc.toZapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to zap config: %w", err)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build base zap logger: %w", err)
	}
	return logger, nil
}
