package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"securecloud/scan/internal/auth"
	"securecloud/scan/internal/checks"
	"securecloud/scan/internal/config"
	"securecloud/scan/internal/handler"
	"securecloud/scan/internal/notify"
	"securecloud/scan/internal/store"
	"securecloud/scan/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	settings, err := config.Load(os.LookupEnv)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(settings.DatabasePath), 0o700); err != nil {
		logger.Error("failed to create database directory", "error", err)
		os.Exit(1)
	}
	scanStore, err := store.New(settings.DatabasePath)
	if err != nil {
		logger.Error("failed to initialise database", "error", err)
		os.Exit(1)
	}
	defer scanStore.Close()
	rootContext, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	suite := checks.NewSuite(settings.NetworkTimeout, settings.CertificateExpiry)
	notificationClient := notify.New(settings.NotificationServiceURL, settings.NotificationTimeout, logger)
	scanWorker := worker.New(scanStore, suite, notificationClient, logger, settings.ScanTimeout, settings.QueueDepth)
	if err := scanWorker.Start(rootContext, settings.WorkerCount); err != nil {
		logger.Error("failed to start scan workers", "error", err)
		os.Exit(1)
	}
	handlerOptions := []handler.Option{
		handler.WithSubmitter(scanWorker),
		handler.WithRateLimit(settings.RequestsPerSecond, settings.RequestBurst),
	}
	if settings.AuthServiceURL != "" {
		handlerOptions = append(handlerOptions, handler.WithVerifier(auth.New(settings.AuthServiceURL, settings.AuthTimeout)))
	}
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(settings.Port),
		Handler:           handler.New(scanStore, logger, handlerOptions...).Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("scan service starting", "port", settings.Port, "workers", settings.WorkerCount, "queue_depth", settings.QueueDepth)
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
	cancelWorkers()
	scanWorker.Wait()
	logger.Info("scan service stopped")
}
