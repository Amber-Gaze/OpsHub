package logger

import (
	"errors"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *zap.SugaredLogger
	initCfg    *options.LoggerConf
	subLoggers []*zap.SugaredLogger
)

func checkConfigDefaults(cfg *options.LoggerConf) {
	if cfg.LogDir == "" {
		cfg.LogDir = "../logs"
	}
	if cfg.LogFileName == "" {
		cfg.LogFileName = "service.log"
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 72 // 默认保留 72 个文件
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7 // days
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Rotation == "" {
		cfg.Rotation = "hour"
	}
	if cfg.Encoding == "" {
		cfg.Encoding = "json"
	}
}

// customTimeEncoder formats the time as "2006-01-02T15:04:05.00" in Shanghai timezone.
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	location, _ := time.LoadLocation("Asia/Shanghai")
	formattedTime := t.In(location).Format("2006-01-02T15:04:05.00")
	enc.AppendString(formattedTime)
}

// customCallerEncoder places the caller information at the end of the log line.
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + caller.TrimmedPath() + "]")
}

// getLogEncoder returns a JSON or Console encoder based on cfg.Encoding.
func getLogEncoder(cfg *options.LoggerConf) zapcore.Encoder {
	encoderConf := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   customCallerEncoder,
	}

	switch cfg.Encoding {
	case "console":
		return zapcore.NewConsoleEncoder(encoderConf)
	default:
		return zapcore.NewJSONEncoder(encoderConf)
	}
}

func getLogWriter(cfg *options.LoggerConf) zapcore.WriteSyncer {
	rotator, err := NewTimeRotator(
		cfg.LogDir,
		cfg.LogFileName,
		cfg.Rotation,
		time.Duration(cfg.MaxAge)*24*time.Hour,
		cfg.MaxBackups,
		cfg.Compress,
	)

	if err != nil {
		// 如果创建 TimeRotator 失败，回退到标准错误输出
		return zapcore.AddSync(os.Stderr)
	}
	return rotator
}

// SetLogLevel sets the minimum log level for the logger and ensures logs are written to the file.
func getLogLevel(cfg *options.LoggerConf) zapcore.Level {
	var zapLevel zapcore.Level
	switch cfg.LogLevel {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}

	return zapLevel
}

// InitLogger initializes the zap logger with the given configuration.
func InitLogger(cfg *options.LoggerConf) error {
	checkConfigDefaults(cfg)

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return err
	}

	// 保存配置副本，供 SubLogger 派生独立文件日志（如 GC 统计）
	initCfg = &*cfg

	core := zapcore.NewCore(
		getLogEncoder(cfg),
		getLogWriter(cfg),
		getLogLevel(cfg),
	)

	// Build logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = zapLogger.Sugar()

	// set to global
	zap.ReplaceGlobals(zapLogger)
	return nil
}

// CurrentLogFileName 返回当前主日志文件名（用于派生子日志文件名）。
func CurrentLogFileName() string {
	if initCfg == nil {
		return ""
	}
	return initCfg.LogFileName
}

// SubLogger 基于当前全局配置，创建一个写入独立文件 <LogDir>/<filename> 的子 logger
// （同样带 TimeRotator 轮转，编码/级别与主日志一致）。用于把高频/独立的日志
// （如 GC 统计）与主日志分开，避免互相污染。
func SubLogger(filename string) (*zap.SugaredLogger, error) {
	if initCfg == nil {
		return nil, errors.New("logger not initialized")
	}
	sub := &options.LoggerConf{
		LogDir:      initCfg.LogDir,
		LogFileName: filename,
		Rotation:    initCfg.Rotation,
		MaxBackups:  initCfg.MaxBackups,
		MaxAge:      initCfg.MaxAge,
		LogLevel:    initCfg.LogLevel,
		Encoding:    initCfg.Encoding,
		Compress:    initCfg.Compress,
	}
	checkConfigDefaults(sub)
	if err := os.MkdirAll(sub.LogDir, 0755); err != nil {
		return nil, err
	}
	core := zapcore.NewCore(getLogEncoder(sub), getLogWriter(sub), getLogLevel(sub))
	zl := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zl.Sugar()
	subLoggers = append(subLoggers, sugar)
	return sugar, nil
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
	for _, s := range subLoggers {
		_ = s.Sync()
	}
}

// Info logs a message at info level.
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof logs a formatted message at info level.
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Debug logs a message at debug level.
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf logs a formatted message at debug level.
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Warn logs a message at warn level.
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Warnf logs a formatted message at warn level.
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Error logs a message at error level.
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf logs a formatted message at error level.
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// GetLogger returns the global logger instance.
// func GetLogger() *zap.SugaredLogger {
// 	return logger
// }

func WithField(key, value string) *zap.SugaredLogger {
	return logger.With(key, value)
}
