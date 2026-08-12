package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

// Init 初始化日志：同时输出到控制台与文件
func Init(level, file string) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), parseLevel(level))

	cores := []zapcore.Core{consoleCore}

	if file != "" {
		if dir := filepath.Dir(file); dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(f), parseLevel(level)))
		}
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	log = logger.Sugar()
}

func parseLevel(level string) zapcore.LevelEnabler {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func ensureLogger() {
	if log == nil {
		Init("info", "")
	}
}

func Debug(args ...interface{})                    { ensureLogger(); log.Debug(args...) }
func Debugf(format string, args ...interface{})  { ensureLogger(); log.Debugf(format, args...) }
func Info(args ...interface{})                    { ensureLogger(); log.Info(args...) }
func Infof(format string, args ...interface{})   { ensureLogger(); log.Infof(format, args...) }
func Warn(args ...interface{})                    { ensureLogger(); log.Warn(args...) }
func Warnf(format string, args ...interface{})   { ensureLogger(); log.Warnf(format, args...) }
func Error(args ...interface{})                   { ensureLogger(); log.Error(args...) }
func Errorf(format string, args ...interface{})  { ensureLogger(); log.Errorf(format, args...) }
func Panic(args ...interface{})                   { ensureLogger(); log.Panic(args...) }
func Panicf(format string, args ...interface{})  { ensureLogger(); log.Panicf(format, args...) }
