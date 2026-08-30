package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/iam/api"
	"github.com/Amber-Gaze/OpsHub/internal/iam/bootstrap"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/redis"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store/mysql"
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
	bootstrap.EnsureAdminFromConfig(context.Background())

	gdb, err := mysql.GetGORM()
	if err != nil {
		logger.Errorf("iam: get gorm db: %v", err)
		Exit(1)
	}
	// 自动建表：user / access_key / service_module（casbin_rule 由 gorm-adapter 自动创建）
	if err := gdb.AutoMigrate(&store.User{}, &store.AccessKey{}, &store.ServiceModule{}); err != nil {
		logger.Errorf("iam: migrate tables: %v", err)
		Exit(1)
	}
	enf, err := casbinx.NewSyncedEnforcer(gdb)
	if err != nil {
		logger.Errorf("iam: casbin init: %v", err)
		Exit(1)
	}

	monitorBase := fallbackMonitorBasePort + authMonitorOffset
	if options.GetAuthMonitoringPort() > 0 {
		monitorBase = options.GetAuthMonitoringPort()
	}
	observability.StartDiagnostics("iam", monitorBase)

	// Redis（可选）：用于登出令牌黑名单。连接失败自动降级不启用。
	var closeRedis func() error
	svc := api.NewService(enf)
	if cache, err := redis.NewCacheFromConfig(); err != nil {
		logger.Warnf("iam: redis disabled (connect failed): %v", err)
	} else if cache != nil {
		svc.SetRedisCache(cache)
		closeRedis = cache.Close
		logger.Infof("iam: redis token-blacklist enabled")
	}

	r := router.New()
	api.RegisterRoutes(r, svc)

	addr := fmt.Sprintf(":%d", options.GetAuthHTTPPort())
	logger.Infof("iam: fasthttp listening on %s", addr)
	err = graceful.RunServer(addr, r.Handler, options.GetShutdownTimeout(), func() error {
		if closeRedis != nil {
			_ = closeRedis()
		}
		if closer, ok := factory.(interface{ Close() error }); ok {
			return closer.Close()
		}
		return nil
	})
	if err != nil {
		logger.Errorf("iam: serve: %v", err)
		Exit(1)
	}
	logger.Infof("iam: graceful shutdown done")
	Exit(0)
}
