package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"securecloud/scan/internal/checks"
	"securecloud/scan/internal/store"
)

type scanner interface {
	Run(context.Context, string) (checks.Report, error)
}

type notifier interface {
	Notify(context.Context, string, string) error
}

type Worker struct {
	store    *store.Store
	scanner  scanner
	notifier notifier
	logger   *slog.Logger
	timeout  time.Duration
	queue    chan string
	wg       sync.WaitGroup
}

func New(scanStore *store.Store, suite scanner, notificationClient notifier, logger *slog.Logger, timeout time.Duration, queueDepth int) *Worker {
	return &Worker{store: scanStore, scanner: suite, notifier: notificationClient, logger: logger, timeout: timeout, queue: make(chan string, queueDepth)}
}

func (w *Worker) Start(ctx context.Context, count int) error {
	if err := w.store.RequeueRunning(ctx); err != nil {
		return err
	}
	pending, err := w.store.ListQueuedIDs(ctx)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		w.wg.Add(1)
		go w.run(ctx)
	}
	for _, id := range pending {
		select {
		case w.queue <- id:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (w *Worker) Wait() { w.wg.Wait() }

func (w *Worker) Enqueue(id string) bool {
	select {
	case w.queue <- id:
		return true
	default:
		return false
	}
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-w.queue:
			w.process(ctx, id)
		}
	}
}

func (w *Worker) process(parent context.Context, id string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			w.logger.Error("scan worker panic recovered", "scan_id", id, "panic", fmt.Sprint(recovered))
			_ = w.store.FailScan(context.Background(), id, "scan worker failed")
		}
	}()
	if err := w.store.MarkRunning(parent, id); err != nil {
		w.logger.Warn("scan is no longer queued", "scan_id", id, "error", err)
		return
	}
	scan, err := w.store.GetScan(parent, id)
	if err != nil {
		w.logger.Error("failed to load queued scan", "scan_id", id, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(parent, w.timeout)
	defer cancel()
	report, err := w.scanner.Run(ctx, scan.Target)
	if err != nil {
		if parent.Err() != nil {
			_ = w.store.RequeueRunning(context.Background())
			return
		}
		if updateErr := w.store.FailScan(context.Background(), id, err.Error()); updateErr != nil {
			w.logger.Error("failed to persist scan failure", "scan_id", id, "error", updateErr)
		}
		return
	}
	if err := w.store.CompleteScan(context.Background(), id, report.Score, report.Findings); err != nil {
		w.logger.Error("failed to persist scan result", "scan_id", id, "error", err)
		return
	}
	w.logger.Info("scan completed", "scan_id", id, "score", report.Score, "finding_count", len(report.Findings))
	if err := w.notifier.Notify(ctx, id, fmt.Sprintf("Security scan completed with score %d/100", report.Score)); err != nil {
		w.logger.Warn("notification delivery failed", "scan_id", id, "error", err)
	}
}
