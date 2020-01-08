package gott

import (
	"net/url"

	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

type lumberjackSink struct {
	*lumberjack.Logger
}

// Sync implements zap.Sink. The remaining methods are implemented
// by the embedded *lumberjack.Logger.
func (lumberjackSink) Sync() error { return nil }

// NewLogger initializes a new zap.Logger with lumberjack support
// to write to log files with rotation.
func NewLogger() *zap.Logger {
	_ = zap.RegisterSink("lumberjack", func(u *url.URL) (zap.Sink, error) {
		return lumberjackSink{
			Logger: &lumberjack.Logger{
				Filename:   u.Opaque,
				MaxSize:    20, // megabytes
				MaxBackups: 30,
				MaxAge:     30, //days
				Compress:   true,
			},
		}, nil
	})

	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"lumberjack:logs/gott.log"}

	logger, _ := config.Build()
	return logger
}
