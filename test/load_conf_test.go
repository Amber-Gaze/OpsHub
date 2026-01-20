package test

import (
	"fmt"
	"testing"

	"github.com/Amber-Gaze/OpsHub/internal/config"
	"github.com/stretchr/testify/assert"
)

var (
	configPath string = "../conf/ops_hub.yaml"
)

func TestLoadConfig(t *testing.T) {
	is := assert.New(t)

	// This is just a placeholder to illustrate where tests for loading configuration would go.
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
		// Handle error
	}

	is.NotNil(conf.Main)
	fmt.Printf("%d %d %d\n", conf.Main.GRPCPort, conf.Main.HTTPPort, conf.Main.MonitoringPort)

	is.NotNil(conf.Logger)
	fmt.Printf("%s/%s %s %d %d\n", conf.Logger.LogDir, conf.Logger.LogFileName, conf.Logger.LogLevel, conf.Logger.MaxSize, conf.Logger.MaxBackups)

	is.NotNil(conf.MySQL)
	fmt.Printf("%s:%d -u %s -p%s\n", conf.MySQL.Host, conf.MySQL.Port, conf.MySQL.User, conf.MySQL.Password)

	is.NotNil(conf.Redis)
	fmt.Printf("%s:%d -p%s\n", conf.Redis.Host, conf.Redis.Port, conf.Redis.Password)
}
