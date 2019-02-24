// Package gcpzap configures the Uber zap logging library for Google Cloud.
package gcpzap

import (
	"net/http"

	"github.com/evanj/gcplogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const encoderName = "stackdriver_json"

func newEncoder(cfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
	return &encoder{zapcore.NewJSONEncoder(cfg)}, nil
}

// NewProductionConfig wraps zap.NewProductionConfig with configuration that works on Google Cloud.
func NewProductionConfig() zap.Config {
	// register the encoder: ignore errors
	_ = zap.RegisterEncoder(encoderName, newEncoder)

	config := zap.NewProductionConfig()
	config.Encoding = encoderName
	config.EncoderConfig.LevelKey = "severity"
	config.EncoderConfig.EncodeLevel = encodeLevel
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.EncodeTime = encodeTime
	return config
}

// NewProduction wraps zap.NewProduction with configuration that works on Google Cloud.
func NewProduction(opts ...zap.Option) (*zap.Logger, error) {
	cfg := NewProductionConfig()
	return cfg.Build(opts...)
}

// WithTraceCore returns a *zap.Logger that will use the trace ID from r, if it is set.
func WithTraceCore(logger *zap.Logger, tracer *gcplogs.Tracer, r *http.Request) *zap.Logger {
	traceID := tracer.FromRequest(r)
	if traceID == "" {
		return logger
	}
	return logger.With(zap.String(gcplogs.TraceKey, traceID))
}

// WithTrace returns a *zap.SugaredLogger that will use the trace ID from r, if it is set.
func WithTrace(
	logger *zap.SugaredLogger, tracer *gcplogs.Tracer, r *http.Request,
) *zap.SugaredLogger {
	return WithTraceCore(logger.Desugar(), tracer, r).Sugar()
}

// Tracer wraps a *zap.Logger to set trace IDs in log messages, if available.
type Tracer struct {
	gcplogs.Tracer
	Logger *zap.Logger
}

// FromRequest returns a *zap.Logger that will use the trace ID from r, if it is set.
func (t *Tracer) FromRequest(r *http.Request) *zap.Logger {
	return WithTraceCore(t.Logger, &t.Tracer, r)
}
