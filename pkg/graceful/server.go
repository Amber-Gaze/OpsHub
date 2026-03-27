package graceful

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
)

const DefaultShutdownTimeout = 20 * time.Second

// RunServer starts a fasthttp server and handles graceful shutdown on SIGINT/SIGTERM.
// It stops accepting new connections and waits for in-flight requests until timeout.
func RunServer(addr string, handler fasthttp.RequestHandler, timeout time.Duration, onShutdown func() error) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}

	srv := &fasthttp.Server{Handler: handler}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := srv.ShutdownWithContext(ctx); err != nil {
			_ = ln.Close()
		}
		if onShutdown != nil {
			if err := onShutdown(); err != nil {
				return err
			}
		}

		err = <-errCh
		if isExpectedServeErr(err) {
			return nil
		}
		return err
	case err = <-errCh:
		if isExpectedServeErr(err) {
			return nil
		}
		return err
	}
}

func isExpectedServeErr(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}
