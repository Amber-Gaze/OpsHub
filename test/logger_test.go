package test

import (
	"testing"

	"github.com/Amber-Gaze/OpsHub/internal/config"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

func TestLogger(t *testing.T) {
	// is := assert.New(t)
	logger.InitLogger(&config.LoggerConf{LogLevel: "info", LogDir: "../logs", LogFileName: "test.log", MaxSize: 100, MaxBackups: 5, MaxAge: 7})
	logger.Infof("%s is ok", "test")
	logger.Debugf("%s is ok", "debug")

	for i := 0; i < 100; i++ {
		logger.Infof("This is info log number %d", i)
	}
}
