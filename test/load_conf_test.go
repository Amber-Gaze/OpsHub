package test

import (
	"fmt"
	"testing"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/stretchr/testify/assert"
)

var (
	configPath = "../build/conf/ops_hub.yaml"
)

func TestLoadConfig(t *testing.T) {
	is := assert.New(t)

	// This is just a placeholder to illustrate where tests for loading configuration would go.
	conf, err := options.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
		// Handle error
	}

	is.NotNil(conf.Main)
	fmt.Printf("%d %d %d\n", conf.Auth.GRPCPort, conf.Auth.HTTPPort, conf.Main.MonitoringPort)

	is.NotNil(conf.Logger)
	fmt.Printf("%s/%s %s %s %d\n", conf.Logger.LogDir, conf.Logger.LogFileName, conf.Logger.LogLevel, conf.Logger.Rotation, conf.Logger.MaxBackups)

	is.NotNil(conf.MySQL)
	fmt.Printf("%s:%d -u %s -p%s\n", conf.MySQL.Host, conf.MySQL.Port, conf.MySQL.User, conf.MySQL.Password)

	is.NotNil(conf.Redis)
	fmt.Printf("%s:%d -p%s\n", conf.Redis.Host, conf.Redis.Port, conf.Redis.Password)
}
