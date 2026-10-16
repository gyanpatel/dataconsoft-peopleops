package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"peopleops/internal/api"
	"peopleops/internal/store"
)

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	secret := env("SESSION_SECRET", "development-only-change-this-secret-before-deploying-123456")
	if len(secret) < 32 {
		logger.Error("SESSION_SECRET must be at least 32 characters")
		os.Exit(1)
	}
	st, e := store.Open()
	if e != nil {
		logger.Error("database initialization failed", "error", e)
		os.Exit(1)
	}
	defer st.DB.Close()
	srv := &api.Server{Store: st, Secret: secret, Origin: env("CORS_ORIGIN", "http://localhost:5173"), Logger: logger}
	httpServer := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("PeopleOps API listening", "addr", httpServer.Addr)
		if e := httpServer.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			logger.Error("server failed", "error", e)
			os.Exit(1)
		}
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e := httpServer.Shutdown(ctx); e != nil {
		logger.Error("graceful shutdown failed", "error", e)
	}
}
