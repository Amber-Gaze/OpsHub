package observability

import (
	"errors"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

const defaultGCInterval = time.Minute

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

// StartGCLogger periodically emits GC related statistics to the shared logger.
func StartGCLogger(service string, interval time.Duration) {
	if interval <= 0 {
		interval = defaultGCInterval
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logger.Infof("[%s] GC logger started, interval=%s", service, interval)

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

			logger.Infof("[%s] gc_stats num_gc=%d pause_total=%s last_pause=%s last_gc=%s heap_alloc=%d heap_idle=%d heap_inuse=%d next_gc=%d", service, stats.NumGC, stats.PauseTotal, lastPause, lastGC, mem.HeapAlloc, mem.HeapIdle, mem.HeapInuse, mem.NextGC)
		}
	}()
}
