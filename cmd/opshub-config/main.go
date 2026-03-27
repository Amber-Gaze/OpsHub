package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/config_center/api"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/etcd"
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

	var svc *api.Service
	if ec := options.GetEtcdConf(); ec != nil && len(ec.Endpoints) > 0 {
		kv, err := etcd.NewConfigKV(ec.Endpoints, ec.Prefix)
		if err != nil {
			logger.Errorf("config-center: etcd connect: %v", err)
			Exit(1)
		}
		defer kv.Close()
		svc = api.NewServiceWithEtcd(kv)
		logger.Infof("config-center: etcd backend endpoints=%v prefix=%q", ec.Endpoints, ec.Prefix)
	} else {
		svc = api.NewService()
		logger.Infof("config-center: in-memory store (set etcd.endpoints to enable persistence)")
	}

	r := router.New()
	api.RegisterRoutes(r, svc)

	addr := fmt.Sprintf(":%d", options.GetConfigCenterHTTPPort())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Errorf("config-center: listen %s: %v", addr, err)
		Exit(1)
	}
	logger.Infof("config-center: fasthttp listening on %s (routes: /configs, /internal/configs)", addr)
	if err := fasthttp.Serve(ln, r.Handler); err != nil {
		logger.Errorf("config-center: serve: %v", err)
		Exit(1)
	}
	Exit(0)
}
