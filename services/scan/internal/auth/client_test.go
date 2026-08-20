package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestVerifyForwardsBearerToken(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "http://auth/verify" || request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("unexpected auth request: %s %q", request.URL, request.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	if err := newClient("http://auth", httpClient).Verify(context.Background(), "Bearer token"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyRejectsUnauthorizedResponse(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	if err := newClient("http://auth", httpClient).Verify(context.Background(), "Bearer bad"); err == nil {
		t.Fatal("Verify() unexpectedly accepted unauthorized response")
	}
}
