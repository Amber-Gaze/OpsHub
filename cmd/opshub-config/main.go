package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/config"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/observability"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

func Exit(code int) {
	logger.Sync()
	time.Sleep(1 * time.Second)
	os.Exit(code)
}

var (
	version     = "0.0.0"
	showVersion = flag.Bool("v", false, "show version")
	configFile  = flag.String("c", "../conf/ops_hub.yaml", "default config file")
)

const (
	fallbackMonitorBasePort   = 8100
	configCenterMonitorOffset = 3
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		return
	}

	conf, err := config.LoadConfig(*configFile)
	if err != nil {
		panic(err)
	}

	conf.Logger.LogFileName = conf.ConfigCenter.LogFileName
	logger.InitLogger(conf.Logger)

	monitorBase := fallbackMonitorBasePort + configCenterMonitorOffset
	if config.GetConfigCenterMonitoringPort() > 0 {
		monitorBase = config.GetConfigCenterMonitoringPort()
	}
	observability.StartDiagnostics("gateway", monitorBase)

	Exit(0)
}
