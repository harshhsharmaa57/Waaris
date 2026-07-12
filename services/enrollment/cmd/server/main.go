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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/postgres"
	"github.com/waaris/waaris/services/enrollment/internal/transport/httpapi"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), config.databaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err = pool.Ping(context.Background()); err != nil {
		slog.Error("database is unavailable", "error", err)
		os.Exit(1)
	}

	tokens, err := application.NewTokenVerifier(config.jwtSecret, config.jwtIssuer)
	if err != nil {
		slog.Error("token configuration failed", "error", err)
		os.Exit(1)
	}

	repository := postgres.New(pool)
	service := application.NewService(repository, repository, application.NewSMTPNotifier(config.smtpAddress, config.smtpFrom))
	server := &http.Server{
		Addr:              config.httpAddr,
		Handler:           httpapi.NewHandlerWithReadiness(service, tokens, pool.Ping).Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	go runLifecycleWorker(service, config.lifecycleTick)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}

type config struct {
	httpAddr      string
	databaseURL   string
	jwtSecret     string
	jwtIssuer     string
	smtpAddress   string
	smtpFrom      string
	lifecycleTick time.Duration
}

func loadConfig() (config, error) {
	lifecycleTick, err := duration("LIFECYCLE_TICK_INTERVAL", time.Minute)
	if err != nil {
		return config{}, err
	}
	result := config{
		httpAddr:      value("HTTP_ADDR", ":8080"),
		databaseURL:   os.Getenv("DATABASE_URL"),
		jwtSecret:     os.Getenv("AUTH_JWT_SECRET"),
		jwtIssuer:     value("AUTH_ACCESS_TOKEN_ISSUER", "waaris-auth"),
		smtpAddress:   value("SMTP_ADDR", "mailpit:1025"),
		smtpFrom:      value("SMTP_FROM", "no-reply@waaris.local"),
		lifecycleTick: lifecycleTick,
	}

	if result.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if result.jwtSecret == "" {
		return config{}, errors.New("AUTH_JWT_SECRET is required")
	}

	return result, nil
}

func value(key, fallback string) string {
	if result := os.Getenv(key); result != "" {
		return result
	}
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive duration")
	}
	return parsed, nil
}

func runLifecycleWorker(service *application.Service, interval time.Duration) {
	if err := service.ProcessLifecycleTick(context.Background()); err != nil {
		slog.Error("initial lifecycle tick failed", "error", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := service.ProcessLifecycleTick(context.Background()); err != nil {
			slog.Error("lifecycle tick failed", "error", err)
		}
	}
}
