package gcpzap

import (
	"regexp"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// The levels must be kept in sync with zap. encodeLevel test should help verify these.
const minLevel = zapcore.DebugLevel
const maxLevel = zapcore.FatalLevel

var logLevelSeverity = [maxLevel - minLevel + 1][]byte{
	[]byte("DEBUG"),
	[]byte("INFO"),
	[]byte("WARNING"),
	[]byte("ERROR"),
	[]byte("CRITICAL"),
	[]byte("ALERT"),
	[]byte("EMERGENCY"),
}

func encodeLevel(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendByteString(logLevelSeverity[l-minLevel])
}

func encodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// RFC3339 is relatively compact and works. See documentation:
	// https://cloud.google.com/logging/docs/agent/configuration#timestamp-processing
	// TODO: Implement our crazy caching scheme?
	enc.AppendString(t.UTC().Format(time.RFC3339Nano))
}

// Wraps zapcore.Encoder to customize stack traces to be picked up by Stackdriver error reporting.
// The following issue might make this unnecessary:
// https://github.com/uber-go/zap/issues/514
type encoder struct {
	jsonEncoder zapcore.Encoder
}

// multiline pattern to match the function name line
var functionNamePattern = regexp.MustCompile(`(?m)^(\S+)$`)

func (s *encoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	if ent.Stack != "" {
		// Make the message look like a real panic, so Stackdriver error reporting picks it up.
		// This used to need the string "panic: " at the beginning, but no longer seems to need it!
		// ent.Message = "panic: " + ent.Message + "\n\ngoroutine 1 [running]:\n"
		ent.Message = ent.Message + "\n\ngoroutine 1 [running]:\n"
		// Trial-and-error: On App Engine Standard go111 the () are needed after function calls
		// zap does not add them, so hack it with a regexp
		replaced := functionNamePattern.ReplaceAllString(ent.Stack, "$1(...)")
		ent.Message += replaced
		ent.Stack = ""
	}
	return s.jsonEncoder.EncodeEntry(ent, fields)
}

func (s *encoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return s.jsonEncoder.AddArray(key, marshaler)
}

func (s *encoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return s.jsonEncoder.AddObject(key, marshaler)
}

func (s *encoder) AddBinary(key string, value []byte) {
	s.jsonEncoder.AddBinary(key, value)
}

func (s *encoder) AddByteString(key string, value []byte) {
	s.jsonEncoder.AddByteString(key, value)
}

func (s *encoder) AddBool(key string, value bool) {
	s.jsonEncoder.AddBool(key, value)
}

func (s *encoder) AddComplex128(key string, value complex128) {
	s.jsonEncoder.AddComplex128(key, value)
}

func (s *encoder) AddComplex64(key string, value complex64) {
	s.jsonEncoder.AddComplex64(key, value)
}

func (s *encoder) AddDuration(key string, value time.Duration) {
	s.jsonEncoder.AddDuration(key, value)
}

func (s *encoder) AddFloat64(key string, value float64) {
	s.jsonEncoder.AddFloat64(key, value)
}

func (s *encoder) AddFloat32(key string, value float32) {
	s.jsonEncoder.AddFloat32(key, value)
}

func (s *encoder) AddInt(key string, value int) {
	s.jsonEncoder.AddInt(key, value)
}

func (s *encoder) AddInt64(key string, value int64) {
	s.jsonEncoder.AddInt64(key, value)
}

func (s *encoder) AddInt32(key string, value int32) {
	s.jsonEncoder.AddInt32(key, value)
}

func (s *encoder) AddInt16(key string, value int16) {
	s.jsonEncoder.AddInt16(key, value)
}

func (s *encoder) AddInt8(key string, value int8) {
	s.jsonEncoder.AddInt8(key, value)
}

func (s *encoder) AddString(key string, value string) {
	s.jsonEncoder.AddString(key, value)
}

func (s *encoder) AddTime(key string, value time.Time) {
	s.jsonEncoder.AddTime(key, value)
}

func (s *encoder) AddUint(key string, value uint) {
	s.jsonEncoder.AddUint(key, value)
}

func (s *encoder) AddUint64(key string, value uint64) {
	s.jsonEncoder.AddUint64(key, value)
}

func (s *encoder) AddUint32(key string, value uint32) {
	s.jsonEncoder.AddUint32(key, value)
}

func (s *encoder) AddUint16(key string, value uint16) {
	s.jsonEncoder.AddUint16(key, value)
}

func (s *encoder) AddUint8(key string, value uint8) {
	s.jsonEncoder.AddUint8(key, value)
}

func (s *encoder) AddUintptr(key string, value uintptr) {
	s.jsonEncoder.AddUintptr(key, value)
}

func (s *encoder) AddReflected(key string, value interface{}) error {
	return s.jsonEncoder.AddReflected(key, value)
}

func (s *encoder) OpenNamespace(key string) {
	s.jsonEncoder.OpenNamespace(key)
}

func (s *encoder) Clone() zapcore.Encoder {
	return &encoder{s.jsonEncoder.Clone()}
}
