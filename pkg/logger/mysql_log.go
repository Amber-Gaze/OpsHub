package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

// ZapGormLogger 将 zap.Logger 适配为 gorm.Logger
type ZapGormLogger struct {
	zapLogger *zap.Logger
}

func NewZapGormLogger() gormlogger.Interface {
	return &ZapGormLogger{zapLogger: zap.S().Named("mysql").Desugar()}
}

func (l *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.zapLogger.Info(msg, zap.Any("data", data))
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.zapLogger.Warn(msg, zap.Any("data", data))
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.zapLogger.Error(msg, zap.Any("data", data))
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		l.zapLogger.Error("gorm trace",
			zap.String("sql", sql),
			zap.Int64("rows_affected", rows),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	} else {
		// 可选：只记录慢查询（例如 > 100ms）
		if elapsed > 100*time.Millisecond {
			l.zapLogger.Warn("slow gorm query",
				zap.String("sql", sql),
				zap.Int64("rows_affected", rows),
				zap.Duration("elapsed", elapsed),
			)
		}
		// 或者记录所有查询（调试用）
		// l.zapLogger.Debug("gorm query",
		//     zap.String("sql", sql),
		//     zap.Int64("rows_affected", rows),
		//     zap.Duration("elapsed", elapsed),
		// )
	}
}
