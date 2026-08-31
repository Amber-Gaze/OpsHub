package observability

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

const defaultGCInterval = time.Minute

func StartDiagnostics(service string, port int) {
	addr := fmt.Sprintf(":%d", port)
	StartPProf(service, addr)
	StartGCLogger(service, time.Minute)
}

// StartPProf launches a HTTP server exposing pprof endpoints on the given address.
// The server runs in a separate goroutine so it does not block the caller.
func StartPProf(service, addr string) {
	if addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	go func() {
		logger.Infof("[%s] pprof server listening on %s", service, addr)
		if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("[%s] pprof server error: %v", service, err)
		}
	}()
}

// StartGCLogger 周期输出 GC 统计，写入独立日志文件（由主日志名派生，如
// ops_hub_config.log → ops_hub_config_gc.log），避免与主业务日志混在一起。
// 若独立文件创建失败则回退到主 logger。
func StartGCLogger(service string, interval time.Duration) {
	if interval <= 0 {
		interval = defaultGCInterval
	}

	gcFileName := "gc.log"
	if base := logger.CurrentLogFileName(); base != "" {
		gcFileName = strings.TrimSuffix(base, ".log") + "_gc.log"
	}
	gcLog, err := logger.SubLogger(gcFileName)
	if err != nil {
		gcLog = nil
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 独立文件优先，失败时回退主 logger
		logf := func(format string, args ...interface{}) {
			if gcLog != nil {
				gcLog.Infof(format, args...)
			} else {
				logger.Infof(format, args...)
			}
		}

		logf("[%s] GC logger started, file=%s interval=%s", service, gcFileName, interval)

		for range ticker.C {
			var stats debug.GCStats
			debug.ReadGCStats(&stats)

			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)

			lastPause := time.Duration(0)
			if len(stats.Pause) > 0 {
				lastPause = stats.Pause[0]
			}

			lastGC := "never"
			if !stats.LastGC.IsZero() {
				lastGC = stats.LastGC.Format(time.RFC3339)
			}

			logf("[%s] gc_stats num_gc=%d pause_total=%s last_pause=%s last_gc=%s heap_alloc=%d heap_idle=%d heap_inuse=%d next_gc=%d", service, stats.NumGC, stats.PauseTotal, lastPause, lastGC, mem.HeapAlloc, mem.HeapIdle, mem.HeapInuse, mem.NextGC)
		}
	}()
}
