package logs

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global structured logger.
var Logger *zap.Logger

type ctxKey struct{}

// Init initialises the global Logger. Call once at startup.
func Init(env, service string) error {
	var cfg zap.Config
	if env == "production" || env == "release" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := cfg.Build(zap.Fields(zap.String("service", service)))
	if err != nil {
		return err
	}
	Logger = l
	return nil
}

// Sync flushes buffered log entries. Call before exit.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// WithContext attaches a logger to a context.
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves the logger from a context, falling back to the global Logger.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}
	if Logger != nil {
		return Logger
	}
	l, _ := zap.NewProduction()
	return l
}

func init() {
	// Ensure Logger is never nil even if Init() is not called.
	if Logger == nil {
		if os.Getenv("GIN_MODE") == "release" {
			Logger, _ = zap.NewProduction()
		} else {
			Logger, _ = zap.NewDevelopment()
		}
	}
}
