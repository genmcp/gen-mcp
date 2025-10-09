package logging

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	LevelDebug     = zapcore.DebugLevel
	LevelInfo      = zapcore.InfoLevel
	LevelWarning   = zapcore.WarnLevel
	LevelError     = zapcore.ErrorLevel
	LevelCritical  = zapcore.DPanicLevel
	LevelAlert     = zapcore.PanicLevel
	LevelEmergency = zapcore.FatalLevel
)

var zapToMCP = map[zapcore.Level]mcp.LoggingLevel{
	LevelDebug:     "debug",
	LevelInfo:      "info",
	LevelWarning:   "warning",
	LevelError:     "error",
	LevelCritical:  "critical",
	LevelAlert:     "alert",
	LevelEmergency: "emergency",
}

type mcpLogger struct {
	ss      *mcp.ServerSession
	ctx     context.Context
	encoder zapcore.Encoder
}

func NewMcpCore(ss *mcp.ServerSession) (zapcore.Core, error) {
	return NewMcpCoreWithContext(context.Background(), ss)
}

func NewMcpCoreWithContext(ctx context.Context, ss *mcp.ServerSession) (zapcore.Core, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if ss == nil {
		return nil, fmt.Errorf("ServerSession cannot be nil")
	}
	return &mcpLogger{
		ss:      ss,
		ctx:     ctx,
		encoder: newMapEncoder(),
	}, nil
}

func (m *mcpLogger) Enabled(zapcore.Level) bool {
	return true // always enabled - the server session decides whether to send the log or not
}

func (m *mcpLogger) With(fields []zapcore.Field) zapcore.Core {
	clone := m.encoder.Clone()

	for _, f := range fields {
		f.AddTo(clone)
	}

	return &mcpLogger{
		ss:      m.ss,
		ctx:     m.ctx, // Preserve context in cloned logger
		encoder: clone,
	}
}

func (m *mcpLogger) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, m)
}

func (m *mcpLogger) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// Create a temporary encoder for additional fields
	tempEncoder := m.encoder.Clone()
	for _, field := range fields {
		field.AddTo(tempEncoder)
	}

	// Get the fields and create a copy to avoid mutation of shared data
	var encoderFields map[string]any
	if mapEnc, ok := tempEncoder.(*mapEncoder); ok {
		encoderFields = mapEnc.fields
	} else {
		// Fallback to empty map if type assertion fails
		encoderFields = make(map[string]any)
	}

	logData := make(map[string]any, len(encoderFields)+3) // +3 for ts, msg, caller

	// Copy existing fields
	for k, v := range encoderFields {
		logData[k] = v
	}

	// Add entry-specific fields
	logData["ts"] = ent.Time
	logData["msg"] = ent.Message

	if ent.Caller.Defined {
		logData["caller"] = ent.Caller.String()
	}

	return m.ss.Log(m.ctx, &mcp.LoggingMessageParams{
		Data:   logData,
		Level:  zapToMCP[ent.Level],
		Logger: ent.LoggerName,
	})
}

func (m *mcpLogger) Sync() error {
	return nil
}

type mapEncoder struct {
	fields        map[string]any
	currentFields map[string]any // pointer to the active map where fields should be added
}

var _ zapcore.Encoder = &mapEncoder{}

func newMapEncoder() *mapEncoder {
	fields := make(map[string]any)
	return &mapEncoder{
		fields:        fields,
		currentFields: fields,
	}
}

// ObjectEncoder methods
func (m *mapEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	arr := &arrayEncoder{items: make([]any, 0)}
	if err := marshaler.MarshalLogArray(arr); err != nil {
		return err
	}
	m.currentFields[key] = arr.items
	return nil
}

func (m *mapEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	obj := newMapEncoder()
	if err := marshaler.MarshalLogObject(obj); err != nil {
		return err
	}
	m.currentFields[key] = obj.fields
	return nil
}

func (m *mapEncoder) AddBinary(key string, value []byte) {
	m.currentFields[key] = base64.StdEncoding.EncodeToString(value)
}

func (m *mapEncoder) AddByteString(key string, value []byte) {
	m.currentFields[key] = string(value)
}

func (m *mapEncoder) AddBool(key string, value bool) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddComplex128(key string, value complex128) {
	m.currentFields[key] = map[string]float64{"real": real(value), "imag": imag(value)}
}

func (m *mapEncoder) AddComplex64(key string, value complex64) {
	m.currentFields[key] = map[string]float64{"real": float64(real(value)), "imag": float64(imag(value))}
}

func (m *mapEncoder) AddDuration(key string, value time.Duration) {
	m.currentFields[key] = value.String()
}

func (m *mapEncoder) AddFloat64(key string, value float64) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddFloat32(key string, value float32) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddInt(key string, value int) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddInt64(key string, value int64) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddInt32(key string, value int32) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddInt16(key string, value int16) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddInt8(key string, value int8) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddString(key string, value string) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddTime(key string, value time.Time) {
	m.currentFields[key] = value.String()
}

func (m *mapEncoder) AddUint(key string, value uint) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddUint64(key string, value uint64) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddUint32(key string, value uint32) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddUint16(key string, value uint16) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddUint8(key string, value uint8) {
	m.currentFields[key] = value
}

func (m *mapEncoder) AddUintptr(key string, value uintptr) {
	m.currentFields[key] = uint64(value)
}

func (m *mapEncoder) AddReflected(key string, value interface{}) error {
	m.currentFields[key] = value
	return nil
}

func (m *mapEncoder) OpenNamespace(key string) {
	// Create a nested map for the namespace
	nested := make(map[string]any)
	m.currentFields[key] = nested
	// Point currentFields to the nested map so subsequent field additions
	// in the same With() call go into this namespace.
	// Note: Clone() resets currentFields to the root, so namespaces don't leak.
	m.currentFields = nested
}

// deepCloneValue performs a deep copy of values to prevent shared references
func deepCloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(val))
		for k, v := range val {
			cloned[k] = deepCloneValue(v)
		}
		return cloned
	case []any:
		cloned := make([]any, len(val))
		for i, v := range val {
			cloned[i] = deepCloneValue(v)
		}
		return cloned
	default:
		// Primitives and other types are safe to copy directly
		return val
	}
}

// Encoder methods
func (m *mapEncoder) Clone() zapcore.Encoder {
	clonedFields := make(map[string]any, len(m.fields))
	for k, v := range m.fields {
		clonedFields[k] = deepCloneValue(v)
	}
	// Always reset currentFields to root to prevent namespace leakage
	// between different logger instances
	return &mapEncoder{
		fields:        clonedFields,
		currentFields: clonedFields,
	}
}

func (m *mapEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Not used by our implementation
	return nil, nil
}

// arrayEncoder for handling array marshaling
type arrayEncoder struct {
	items []any
}

var _ zapcore.ArrayEncoder = &arrayEncoder{}

func (a *arrayEncoder) AppendArray(marshaler zapcore.ArrayMarshaler) error {
	arr := &arrayEncoder{items: make([]any, 0)}
	if err := marshaler.MarshalLogArray(arr); err != nil {
		return err
	}
	a.items = append(a.items, arr.items)
	return nil
}

func (a *arrayEncoder) AppendObject(marshaler zapcore.ObjectMarshaler) error {
	obj := newMapEncoder()
	if err := marshaler.MarshalLogObject(obj); err != nil {
		return err
	}
	a.items = append(a.items, obj.fields)
	return nil
}

func (a *arrayEncoder) AppendReflected(value interface{}) error {
	a.items = append(a.items, value)
	return nil
}

func (a *arrayEncoder) AppendBool(value bool) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendByteString(value []byte) {
	a.items = append(a.items, string(value))
}

func (a *arrayEncoder) AppendComplex128(value complex128) {
	a.items = append(a.items, map[string]float64{"real": real(value), "imag": imag(value)})
}

func (a *arrayEncoder) AppendComplex64(value complex64) {
	a.items = append(a.items, map[string]float64{"real": float64(real(value)), "imag": float64(imag(value))})
}

func (a *arrayEncoder) AppendDuration(value time.Duration) {
	a.items = append(a.items, value.String())
}

func (a *arrayEncoder) AppendFloat64(value float64) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendFloat32(value float32) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendInt(value int) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendInt64(value int64) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendInt32(value int32) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendInt16(value int16) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendInt8(value int8) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendString(value string) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendTime(value time.Time) {
	a.items = append(a.items, value.String())
}

func (a *arrayEncoder) AppendUint(value uint) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendUint64(value uint64) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendUint32(value uint32) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendUint16(value uint16) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendUint8(value uint8) {
	a.items = append(a.items, value)
}

func (a *arrayEncoder) AppendUintptr(value uintptr) {
	a.items = append(a.items, uint64(value))
}
