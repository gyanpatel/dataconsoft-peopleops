package main

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	if err := os.MkdirAll("logs", 0750); err != nil {
		log.Printf("cannot create logs directory: %v", err)
		os.Exit(1)
	}
	logPath := filepath.Join("logs", "peopleops-"+time.Now().Format("2006-01-02")+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		log.Printf("cannot open daily log file %s: %v", logPath, err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{AddSource: true}))
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
	// The configured Windows time-zone label is mapped to its IANA equivalent
	// because Go's time.LoadLocation uses the IANA time-zone database.
	var configuredZone string
	if err := st.DB.QueryRow(context.Background(), `SELECT timezone_name FROM dcs_app_timezone WHERE active=true AND is_default=true LIMIT 1`).Scan(&configuredZone); err != nil {
		logger.Error("failed to read configured time zone", "error", err)
		os.Exit(1)
	}
	zoneID := strings.TrimSpace(configuredZone)
	if strings.EqualFold(zoneID, "Arabian Standard Time") {
		zoneID = "Asia/Dubai"
	}
	loc, err := time.LoadLocation(zoneID)
	if err != nil {
		logger.Error("configured time zone is invalid", "configuredZone", configuredZone, "error", err)
		os.Exit(1)
	}
	time.Local = loc
	logger.Info("application time zone configured", "timezone", configuredZone, "ianaTimezone", zoneID)
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
