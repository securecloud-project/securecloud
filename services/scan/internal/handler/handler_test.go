package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"securecloud/scan/internal/store"
)

type fakeVerifier struct{ err error }

func (v fakeVerifier) Verify(context.Context, string) error { return v.err }

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	scanStore, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	t.Cleanup(func() { _ = scanStore.Close() })
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return New(scanStore, logger).Router()
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestHealthEndpoints(t *testing.T) {
	router := newTestRouter(t)
	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/healthz", body: `{"status":"ok"}`},
		{path: "/readyz", body: `{"status":"ready"}`},
	} {
		response := performRequest(router, http.MethodGet, test.path, "")
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", test.path, response.Code)
		}
		if strings.TrimSpace(response.Body.String()) != test.body {
			t.Errorf("GET %s body = %q, want %q", test.path, response.Body.String(), test.body)
		}
	}
}

func TestScanLifecycle(t *testing.T) {
	router := newTestRouter(t)
	createdResponse := performRequest(router, http.MethodPost, "/scan", `{"target":" example.com "}`)
	if createdResponse.Code != http.StatusAccepted {
		t.Fatalf("POST /scan status = %d, want 202", createdResponse.Code)
	}
	var created store.Scan
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if created.Target != "example.com" || created.Status != store.StatusQueued {
		t.Fatalf("POST /scan = %+v", created)
	}
	getResponse := performRequest(router, http.MethodGet, "/scan/"+created.ID, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /scan/{id} status = %d, want 200", getResponse.Code)
	}
	listResponse := performRequest(router, http.MethodGet, "/scans", "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("GET /scans status = %d, want 200", listResponse.Code)
	}
	var scans []store.Scan
	if err := json.NewDecoder(listResponse.Body).Decode(&scans); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(scans) != 1 || scans[0].ID != created.ID {
		t.Fatalf("GET /scans = %+v", scans)
	}
}

func TestCreateScanRejectsInvalidRequests(t *testing.T) {
	router := newTestRouter(t)
	for _, body := range []string{
		`{"target":""}`,
		`{"target":"   "}`,
		`hello`,
		`{"target":"example.com","extra":true}`,
		`{"target":"https://example.com"}`,
		`{"target":"127.0.0.1"}`,
		`{"target":"example.com"} {"target":"other.example"}`,
	} {
		response := performRequest(router, http.MethodPost, "/scan", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("POST /scan body %q status = %d, want 400", body, response.Code)
		}
	}
}

func TestGetScanNotFound(t *testing.T) {
	response := performRequest(newTestRouter(t), http.MethodGet, "/scan/does-not-exist", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("GET missing scan status = %d, want 404", response.Code)
	}
}

func TestScanRoutesRequireValidBearerTokenWhenVerifierConfigured(t *testing.T) {
	scanStore, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer scanStore.Close()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	router := New(scanStore, logger, WithVerifier(fakeVerifier{})).Router()

	missing := performRequest(router, http.MethodGet, "/scans", "")
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer status = %d, want 401", missing.Code)
	}
	request := httptest.NewRequest(http.MethodGet, "/scans", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("verified bearer status = %d, want 200", response.Code)
	}
}

func TestTokenBucketEnforcesBurst(t *testing.T) {
	bucket := newTokenBucket(1, 2)
	now := time.Now()
	if !bucket.Allow(now) || !bucket.Allow(now) {
		t.Fatal("initial burst was rejected")
	}
	if bucket.Allow(now) {
		t.Fatal("request beyond burst was accepted")
	}
	if !bucket.Allow(now.Add(time.Second)) {
		t.Fatal("refilled token was rejected")
	}
}
