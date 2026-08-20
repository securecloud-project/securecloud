package notify

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestNotify(t *testing.T) {
	received := false
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/notifications" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		received = true
		return &http.Response{StatusCode: http.StatusCreated, Status: "201 Created", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	client := newClient("http://notification", httpClient, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := client.Notify(context.Background(), "scan-1", "complete"); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if !received {
		t.Fatal("notification was not received")
	}
}

func TestNotificationFailureIsReturned(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	client := newClient("http://notification", httpClient, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := client.Notify(context.Background(), "scan-1", "complete"); err == nil {
		t.Fatal("Notify() unexpectedly succeeded")
	}
}
