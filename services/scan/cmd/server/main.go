package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"securecloud/scan/internal/handler"
	"securecloud/scan/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := getEnv("PORT", "8080")
	dbPath := getEnv("SCAN_DB_PATH", "./data/scan.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		logger.Error("failed to create database directory", "error", err)
		os.Exit(1)
	}
	scanStore, err := store.New(dbPath)
	if err != nil {
		logger.Error("failed to initialise database", "error", err)
		os.Exit(1)
	}
	defer scanStore.Close()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler.New(scanStore, logger).Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("scan service starting", "port", port, "database", dbPath)
		serverErrors <- server.ListenAndServe()
	}()
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalContext.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			return
		}
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return
	}
	logger.Info("scan service stopped")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
