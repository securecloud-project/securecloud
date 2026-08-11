package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	testStore, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = testStore.Close() })
	return testStore
}

func TestCreateAndGetScan(t *testing.T) {
	testStore := newTestStore(t)
	created, err := testStore.CreateScan(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("CreateScan() error = %v", err)
	}
	if created.ID == "" || created.Target != "example.com" || created.Status != StatusQueued {
		t.Fatalf("CreateScan() = %+v", created)
	}
	if created.Score != 0 || len(created.Findings) != 0 {
		t.Fatalf("new scan has score %d and %d findings", created.Score, len(created.Findings))
	}
	found, err := testStore.GetScan(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetScan() error = %v", err)
	}
	if found.ID != created.ID || found.Target != created.Target {
		t.Fatalf("GetScan() = %+v, want %+v", found, created)
	}
}

func TestGetScanNotFound(t *testing.T) {
	_, err := newTestStore(t).GetScan(context.Background(), "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetScan() error = %v, want sql.ErrNoRows", err)
	}
}

func TestListScans(t *testing.T) {
	testStore := newTestStore(t)
	for _, target := range []string{"example.com", "google.com"} {
		if _, err := testStore.CreateScan(context.Background(), target); err != nil {
			t.Fatalf("CreateScan(%q) error = %v", target, err)
		}
	}
	scans, err := testStore.ListScans(context.Background())
	if err != nil {
		t.Fatalf("ListScans() error = %v", err)
	}
	if len(scans) != 2 {
		t.Fatalf("ListScans() returned %d scans, want 2", len(scans))
	}
}
