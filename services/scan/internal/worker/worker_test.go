package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"securecloud/scan/internal/checks"
	"securecloud/scan/internal/store"
)

type fakeScanner struct {
	report checks.Report
	err    error
}

func (s fakeScanner) Run(context.Context, string) (checks.Report, error) { return s.report, s.err }

type fakeNotifier struct{ err error }

func (n fakeNotifier) Notify(context.Context, string, string) error { return n.err }

func TestWorkerCompletesWhenNotificationFails(t *testing.T) {
	testStore, _ := store.New(filepath.Join(t.TempDir(), "scan.db"))
	defer testStore.Close()
	ctx, cancel := context.WithCancel(context.Background())
	worker := New(testStore, fakeScanner{report: checks.Report{Score: 85}}, fakeNotifier{err: errors.New("offline")}, slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Second, 1)
	if err := worker.Start(ctx, 1); err != nil {
		t.Fatal(err)
	}
	scan, _ := testStore.CreateScan(context.Background(), "example.com")
	if !worker.Enqueue(scan.ID) {
		t.Fatal("Enqueue() failed")
	}
	waitForStatus(t, testStore, scan.ID, store.StatusComplete)
	cancel()
	worker.Wait()
}

func TestWorkerMarksFailedScan(t *testing.T) {
	testStore, _ := store.New(filepath.Join(t.TempDir(), "scan.db"))
	defer testStore.Close()
	ctx, cancel := context.WithCancel(context.Background())
	worker := New(testStore, fakeScanner{err: errors.New("timeout")}, fakeNotifier{}, slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Second, 1)
	if err := worker.Start(ctx, 1); err != nil {
		t.Fatal(err)
	}
	scan, _ := testStore.CreateScan(context.Background(), "example.com")
	if !worker.Enqueue(scan.ID) {
		t.Fatal("Enqueue() failed")
	}
	waitForStatus(t, testStore, scan.ID, store.StatusFailed)
	cancel()
	worker.Wait()
}

func waitForStatus(t *testing.T, testStore *store.Store, id string, wanted store.ScanStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		scan, err := testStore.GetScan(context.Background(), id)
		if err == nil && scan.Status == wanted {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("scan did not reach %s", wanted)
}
