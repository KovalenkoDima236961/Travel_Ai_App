package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var (
	once sync.Once
	root *zap.Logger
)

// InitLogger initialises the process logger.
func InitLogger() *zap.Logger {
	once.Do(func() {
		root = buildLogger()
	})
	return root
}

// FromCtx returns a logger stored in context.
func FromCtx(ctx context.Context) (*zap.Logger, bool) {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l, true
	}
	return nil, false
}

// WithCtx stores a logger in context.
func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		if lp == l {
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}

func buildLogger() *zap.Logger {
	stdout := zapcore.AddSync(os.Stdout)
	stderr := zapcore.AddSync(os.Stderr)

	stdoutCfg := zap.NewDevelopmentEncoderConfig()
	stdoutCfg.StacktraceKey = "stack"
	stdoutCfg.CallerKey = "caller"
	stdoutCfg.LevelKey = "lvl"
	stdoutCfg.MessageKey = "msg"
	stdoutCfg.EncodeCaller = zapcore.ShortCallerEncoder
	stdoutCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	stdoutCfg.EncodeDuration = zapcore.MillisDurationEncoder
	stdoutCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(stdoutCfg)

	debugLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l < zapcore.ErrorLevel
	})

	errorLevel := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.ErrorLevel
	})

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, debugLevel),
		zapcore.NewCore(consoleEncoder, stderr, errorLevel),
	)

	return zap.New(core, zap.AddStacktrace(zapcore.ErrorLevel), zap.AddCaller()).With(zap.String("env", "development"))
}
