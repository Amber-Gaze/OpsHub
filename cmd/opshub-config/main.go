package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
	"github.com/Amber-Gaze/OpsHub/pkg/observability"
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

	conf, err := options.LoadConfig(*configFile)
	if err != nil {
		panic(err)
	}

	conf.Logger.LogFileName = conf.ConfigCenter.LogFileName
	logger.InitLogger(conf.Logger)

	monitorBase := fallbackMonitorBasePort + configCenterMonitorOffset
	if options.GetConfigCenterMonitoringPort() > 0 {
		monitorBase = options.GetConfigCenterMonitoringPort()
	}
	observability.StartDiagnostics("config-center", monitorBase)

	Exit(0)
}
