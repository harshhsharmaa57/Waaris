package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/waaris/waaris/platform/health"
)

const serviceName = "notification"

func main() {
	mux := http.NewServeMux()
	health.NewHandler(serviceName).Register(mux)
	server := &http.Server{
		Addr:              address(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "service", serviceName, "error", err)
			os.Exit(1)
		}
	}()
	<-shutdownSignal()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "service", serviceName, "error", err)
	}
}

func address() string {
	if value := os.Getenv("HTTP_ADDR"); value != "" {
		return value
	}
	return ":8080"
}
func shutdownSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	return signals
}
