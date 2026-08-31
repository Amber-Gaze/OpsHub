package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/config_center/api"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/etcd"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/redis"
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
	fallbackMonitorBasePort   = 8201
	configCenterMonitorOffset = 2
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

	var (
		svc        *api.Service
		closeEtcd  func() error
		closeRedis func() error
	)
	if ec := options.GetEtcdConf(); ec != nil && len(ec.Endpoints) > 0 {
		kv, err := etcd.NewConfigKV(ec.Endpoints, ec.Prefix)
		if err != nil {
			logger.Errorf("config-center: etcd connect: %v", err)
			Exit(1)
		}
		svc = api.NewServiceWithEtcd(kv)
		closeEtcd = kv.Close
		// 审计历史存到 sibling prefix（如 /opshub/config-audit），与配置本体互不污染。
		// 必须是配置前缀的 sibling 而非子前缀：若为子前缀（/opshub/config/audit），
		// ConfigKV.List 的 WithPrefix(配置前缀) 会把审计记录一并列出，导致树/列表重复。
		auditPrefix := strings.TrimRight(ec.Prefix, "/") + "-audit"
		if auditKV, err := etcd.NewConfigKV(ec.Endpoints, auditPrefix); err == nil {
			svc.SetAuditKV(auditKV)
			logger.Infof("config-center: audit history backend prefix=%q", auditPrefix)
		} else {
			logger.Warnf("config-center: audit history disabled: %v", err)
		}
		logger.Infof("config-center: etcd backend endpoints=%v prefix=%q", ec.Endpoints, ec.Prefix)
	} else {
		svc = api.NewService()
		logger.Infof("config-center: in-memory store (set etcd.endpoints to enable persistence)")
	}

	// Redis L1 读缓存（可选，混存：etcd 主存 + Redis 缓存）。连接失败自动降级不启用。
	if cache, err := redis.NewCacheFromConfig(); err != nil {
		logger.Warnf("config-center: redis cache disabled (connect failed): %v", err)
	} else if cache != nil {
		svc.SetRedisCache(cache)
		closeRedis = cache.Close
		logger.Infof("config-center: redis L1 cache enabled (ttl=%s)", cache.TTL())
	}

	r := router.New()
	api.RegisterRoutes(r, svc)

	addr := fmt.Sprintf(":%d", options.GetConfigCenterHTTPPort())
	logger.Infof("config-center: fasthttp listening on %s (routes: /configs, /internal/configs)", addr)
	err = graceful.RunServer(addr, r.Handler, options.GetShutdownTimeout(), func() error {
		if closeRedis != nil {
			_ = closeRedis()
		}
		if closeEtcd != nil {
			return closeEtcd()
		}
		return nil
	})
	if err != nil {
		logger.Errorf("config-center: serve: %v", err)
		Exit(1)
	}
	logger.Infof("config-center: graceful shutdown done")
	Exit(0)
}
