package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/gateway/api"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/utils"
	"github.com/Amber-Gaze/OpsHub/pkg/graceful"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
	"github.com/Amber-Gaze/OpsHub/pkg/observability"
	"github.com/fasthttp/router"
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
	fallbackMonitorBasePort = 8100
	gatewayMonitorOffset    = 2
	defaultRateLimitRPS     = 100
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
	conf.Logger.LogFileName = conf.Gateway.LogFileName
	logger.InitLogger(conf.Logger)

	monitorBase := fallbackMonitorBasePort + gatewayMonitorOffset
	if options.GetGatewayMonitoringPort() > 0 {
		monitorBase = options.GetGatewayMonitoringPort()
	}
	observability.StartDiagnostics("gateway", monitorBase)

	authBaseURL := utils.GetGatewayAuthBaseURL()
	configCenterBaseURL := utils.GetGatewayConfigCenterBaseURL()
	svc := api.NewService(authBaseURL, configCenterBaseURL)

	r := router.New()
	api.RegisterRoutes(r, svc, api.RoutesConfig{
		AuthBaseURL:  authBaseURL,
		LoginPath:    "/login",
		RateLimitRPS: defaultRateLimitRPS,
	})

	addr := fmt.Sprintf(":%d", options.GetGatewayHTTPPort())
	logger.Infof("gateway: fasthttp listening on %s (auth=%s config=%s)", addr, authBaseURL, configCenterBaseURL)
	err = graceful.RunServer(addr, r.Handler, graceful.DefaultShutdownTimeout, nil)
	if err != nil {
		logger.Errorf("gateway: serve: %v", err)
		Exit(1)
	}
	logger.Infof("gateway: graceful shutdown done")
	Exit(0)
}
