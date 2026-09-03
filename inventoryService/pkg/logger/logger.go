package logger

import (
	"context"
	"go.elastic.co/ecszap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"os"
)

var Log *zap.Logger

func Init() {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
	Log = zap.New(core, zap.AddCaller())
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

func WithTraceID(ctx context.Context) zap.Field {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return zap.String("trace.id", span.SpanContext().TraceID().String())
	}
	return zap.Skip()
}
