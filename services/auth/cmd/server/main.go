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
	"github.com/waaris/waaris/services/auth/internal/application"
	"github.com/waaris/waaris/services/auth/internal/infrastructure/postgres"
	"github.com/waaris/waaris/services/auth/internal/transport/httpapi"
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
	tokens, err := application.NewTokenManager(config.jwtSecret, "waaris-auth", config.accessTTL)
	if err != nil {
		slog.Error("token configuration failed", "error", err)
		os.Exit(1)
	}
	service := application.NewService(postgres.New(pool), tokens, config.refreshTTL)
	server := &http.Server{Addr: config.httpAddr, Handler: httpapi.NewHandler(service).Router(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
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
	httpAddr, databaseURL, jwtSecret string
	accessTTL, refreshTTL            time.Duration
}

func loadConfig() (config, error) {
	accessTTL, err := duration("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return config{}, err
	}
	refreshTTL, err := duration("AUTH_REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return config{}, err
	}
	result := config{httpAddr: value("HTTP_ADDR", ":8080"), databaseURL: os.Getenv("DATABASE_URL"), jwtSecret: os.Getenv("AUTH_JWT_SECRET"), accessTTL: accessTTL, refreshTTL: refreshTTL}
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
