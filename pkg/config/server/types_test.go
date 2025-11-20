package server

import (
	"testing"

	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestGetBaseLogger(t *testing.T) {
	tt := map[string]struct {
		runtime        *ServerRuntime
		expectNoop     bool
		expectConsole  bool
		expectCustom   bool
		customLogLevel zapcore.Level
	}{
		"nil runtime returns noop": {
			runtime:    nil,
			expectNoop: true,
		},
		"no logging config defaults to console logger": {
			runtime: &ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &StreamableHTTPConfig{
					Port: 8080,
				},
			},
			expectConsole: true,
		},
		"with logging config uses custom logger": {
			runtime: &ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &StreamableHTTPConfig{
					Port: 8080,
				},
				LoggingConfig: &logging.LoggingConfig{
					Level:    "warn",
					Encoding: "console",
				},
			},
			expectCustom:   true,
			customLogLevel: zapcore.WarnLevel,
		},
		"with json encoding uses custom logger": {
			runtime: &ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &StreamableHTTPConfig{
					Port: 8080,
				},
				LoggingConfig: &logging.LoggingConfig{
					Level:    "info",
					Encoding: "json",
				},
			},
			expectCustom:   true,
			customLogLevel: zapcore.InfoLevel,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			logger := tc.runtime.GetBaseLogger()
			assert.NotNil(t, logger)

			// Test that the logger is not a no-op by checking if it can log
			// We'll verify the logger type indirectly by checking if it's enabled
			if tc.expectNoop {
				// No-op logger should always return false for Enabled checks
				assert.False(t, logger.Core().Enabled(zapcore.DebugLevel))
				assert.False(t, logger.Core().Enabled(zapcore.InfoLevel))
				assert.False(t, logger.Core().Enabled(zapcore.WarnLevel))
			} else if tc.expectConsole {
				// Console logger should be enabled for info level and above
				assert.False(t, logger.Core().Enabled(zapcore.DebugLevel))
				assert.True(t, logger.Core().Enabled(zapcore.InfoLevel))
				assert.True(t, logger.Core().Enabled(zapcore.WarnLevel))
			} else if tc.expectCustom {
				// Custom logger should respect the configured level
				// If level is warn, info should be disabled
				// If level is info, info should be enabled
				expectedInfoEnabled := tc.customLogLevel <= zapcore.InfoLevel
				assert.Equal(t, expectedInfoEnabled, logger.Core().Enabled(zapcore.InfoLevel),
					"Info level enabled check failed for level %v", tc.customLogLevel)
				// Warn level should always be enabled if custom level is warn or below
				expectedWarnEnabled := tc.customLogLevel <= zapcore.WarnLevel
				assert.Equal(t, expectedWarnEnabled, logger.Core().Enabled(zapcore.WarnLevel),
					"Warn level enabled check failed for level %v", tc.customLogLevel)
			}

			// Verify logger can actually log (not a no-op)
			// We verify this by checking that the logger respects levels appropriately
			// For console/default loggers, info level should be enabled
			// For custom loggers, they should respect their configured level
			if !tc.expectNoop {
				// Default console logger should be enabled for info level
				if tc.expectConsole {
					assert.True(t, logger.Core().Enabled(zapcore.InfoLevel))
				}
				// Custom loggers are already verified above
			}
		})
	}
}

func TestGetBaseLoggerMultipleCalls(t *testing.T) {
	// Test that GetBaseLogger is idempotent and returns the same logger instance
	runtime := &ServerRuntime{
		TransportProtocol: TransportProtocolStreamableHttp,
		StreamableHTTPConfig: &StreamableHTTPConfig{
			Port: 8080,
		},
	}

	logger1 := runtime.GetBaseLogger()
	logger2 := runtime.GetBaseLogger()

	// Should return the same logger instance (cached)
	assert.Equal(t, logger1, logger2)
}

func TestGetBaseLoggerWithInvalidConfig(t *testing.T) {
	// Test that invalid logging config falls back to console logger
	runtime := &ServerRuntime{
		TransportProtocol: TransportProtocolStreamableHttp,
		StreamableHTTPConfig: &StreamableHTTPConfig{
			Port: 8080,
		},
		LoggingConfig: &logging.LoggingConfig{
			Level: "invalid-level", // This should cause an error
		},
	}

	// Should not panic and should return a logger (fallback to console)
	logger := runtime.GetBaseLogger()
	assert.NotNil(t, logger)
	// Should be enabled for info level (console logger fallback)
	assert.True(t, logger.Core().Enabled(zapcore.InfoLevel))
}
