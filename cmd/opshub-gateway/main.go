package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/gateway/api"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/utils"
	"github.com/Amber-Gaze/OpsHub/pkg/graceful"
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
	fallbackMonitorBasePort = 8001
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

	// 运控台前端静态页面（可选）：以网关为统一入口，同源访问避免跨域
	if serveWeb(r) {
		logger.Infof("gateway: serving web console (open http://127.0.0.1:%d/)", options.GetGatewayHTTPPort())
	}

	addr := fmt.Sprintf(":%d", options.GetGatewayHTTPPort())
	logger.Infof("gateway: fasthttp listening on %s (auth=%s config=%s)", addr, authBaseURL, configCenterBaseURL)
	err = graceful.RunServer(addr, r.Handler, options.GetShutdownTimeout(), nil)
	if err != nil {
		logger.Errorf("gateway: serve: %v", err)
		Exit(1)
	}
	logger.Infof("gateway: graceful shutdown done")
	Exit(0)
}

// serveWeb 注册运控台前端页面路由（/ 返回 index.html，/static/* 返回静态资源）。
func serveWeb(r *router.Router) bool {
	webDir, ok := resolveWebDir()
	if !ok {
		return false
	}
	index := filepath.Join(webDir, "index.html")

	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		ctx.SendFile(index)
	})
	r.GET("/static/{filepath:*}", func(ctx *fasthttp.RequestCtx) {
		p := strings.TrimPrefix(string(ctx.Path()), "/static/")
		p = path.Clean("/" + p)[1:]
		if p == "" || strings.Contains(p, "..") {
			ctx.Error("not found", fasthttp.StatusNotFound)
			return
		}
		ctx.SendFile(filepath.Join(webDir, "static", p))
	})
	return true
}

// resolveWebDir 定位前端目录：优先当前工作目录 ./web（开发态），
// 其次可执行文件上级的 web（部署态：output/bin 下的二进制 → output/web）。
func resolveWebDir() (string, bool) {
	if p, err := filepath.Abs("web"); err == nil {
		if st, err := os.Stat(filepath.Join(p, "index.html")); err == nil && !st.IsDir() {
			return p, true
		}
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "web")
		if st, err := os.Stat(filepath.Join(p, "index.html")); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}
