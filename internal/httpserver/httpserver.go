// Package httpserver provides a standard graceful-shutdown wrapper around
// net/http.Server so every gin-based service in this repo drains in-flight
// requests instead of being hard-killed on SIGTERM (e.g. `docker compose
// stop`, k8s pod termination).
package httpserver

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Run starts an HTTP server on addr and blocks until it receives
// SIGINT/SIGTERM, at which point it stops accepting new connections and
// waits (up to 15s) for in-flight requests to finish before returning.
func Run(addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Println("shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
