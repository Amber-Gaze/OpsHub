package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/iam/api"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store/mysql"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
	"github.com/Amber-Gaze/OpsHub/pkg/observability"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
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
	authMonitorOffset       = 1
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
	conf.Logger.LogFileName = conf.Auth.LogFileName
	logger.InitLogger(conf.Logger)

	factory, err := mysql.GetMySQLFactoryOr(options.GetMySQLOptionsFromConfig())
	if err != nil {
		logger.Errorf("iam: mysql store init failed: %v", err)
		Exit(1)
	}
	store.SetClient(factory)

	monitorBase := fallbackMonitorBasePort + authMonitorOffset
	if options.GetAuthMonitoringPort() > 0 {
		monitorBase = options.GetAuthMonitoringPort()
	}
	observability.StartDiagnostics("iam", monitorBase)

	r := router.New()
	svc := api.NewService()
	api.RegisterRoutes(r, svc)

	addr := fmt.Sprintf(":%d", options.GetAuthHTTPPort())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Errorf("iam: listen %s: %v", addr, err)
		Exit(1)
	}
	logger.Infof("iam: fasthttp listening on %s", addr)
	if err := fasthttp.Serve(ln, r.Handler); err != nil {
		logger.Errorf("iam: serve: %v", err)
		Exit(1)
	}
	Exit(0)
}
