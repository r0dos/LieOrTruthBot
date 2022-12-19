package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLogger *zap.Logger

func Initialize() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	logFile, _ := os.OpenFile("log.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(config),
			zap.CombineWriteSyncers(zapcore.AddSync(logFile), os.Stdout),
			zapcore.DebugLevel),
	)
	zapLogger = zap.New(core)
}

func With(fields ...zap.Field) *zap.Logger {
	return zapLogger.With(fields...)
}

func Sugar() *zap.SugaredLogger {
	return zapLogger.Sugar()
}

func Sync() {
	_ = zapLogger.Sync()
}

func Log(lvl zapcore.Level, message string, fields ...zap.Field) {
	zapLogger.Log(lvl, message, fields...)
}

func Info(message string, fields ...zap.Field) {
	zapLogger.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	zapLogger.Debug(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	zapLogger.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	zapLogger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	zapLogger.Fatal(message, fields...)
}

type ctxLogger struct{}

// ContextWithLogger adds logger to context
func ContextWithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxLogger{}, l)
}

// LoggerFromContext returns logger from context
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxLogger{}).(*zap.Logger); ok {
		return l
	}
	return zap.L()
}
