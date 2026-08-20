package checks

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestValidHSTS(t *testing.T) {
	if !validHSTS("max-age=31536000; includeSubDomains") {
		t.Fatal("valid HSTS header was rejected")
	}
	for _, value := range []string{"", "includeSubDomains", "max-age=0", "max-age=abc", "max-age=-1", "max-age=123junk"} {
		if validHSTS(value) {
			t.Errorf("invalid HSTS header %q was accepted", value)
		}
	}
}

func TestValidXContentTypeOptions(t *testing.T) {
	for _, value := range []string{"nosniff", "NoSniff", "other, nosniff"} {
		if !validXContentTypeOptions(value) {
			t.Errorf("valid X-Content-Type-Options %q was rejected", value)
		}
	}
	for _, value := range []string{"", "not-nosniff", "nosniff-extra"} {
		if validXContentTypeOptions(value) {
			t.Errorf("invalid X-Content-Type-Options %q was accepted", value)
		}
	}
}

func TestRedirectStatus(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		if !isRedirectStatus(status) {
			t.Errorf("redirect status %d was rejected", status)
		}
	}
	for _, status := range []int{200, 300, 304, 305, 306} {
		if isRedirectStatus(status) {
			t.Errorf("non-redirect status %d was accepted", status)
		}
	}
}

func TestCertificateValidity(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name        string
		certificate x509.Certificate
		points      int
		valid       bool
		findings    int
	}{
		{name: "valid", certificate: x509.Certificate{NotBefore: now.Add(-time.Hour), NotAfter: now.Add(90 * 24 * time.Hour)}, points: 40, valid: true},
		{name: "near expiry", certificate: x509.Certificate{NotBefore: now.Add(-time.Hour), NotAfter: now.Add(10 * 24 * time.Hour)}, points: 30, valid: true, findings: 1},
		{name: "expired", certificate: x509.Certificate{NotBefore: now.Add(-48 * time.Hour), NotAfter: now.Add(-time.Hour)}, valid: false, findings: 1},
		{name: "not yet valid", certificate: x509.Certificate{NotBefore: now.Add(time.Hour), NotAfter: now.Add(90 * 24 * time.Hour)}, valid: false, findings: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			points, findings, valid := certificateValidity(&test.certificate, now, 30*24*time.Hour)
			if points != test.points || valid != test.valid || len(findings) != test.findings {
				t.Fatalf("certificateValidity() = %d, %d findings, %v", points, len(findings), valid)
			}
		})
	}
}

func TestHeaderScoring(t *testing.T) {
	suite := &Suite{httpsClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{
			"Strict-Transport-Security": []string{"max-age=31536000"},
			"X-Content-Type-Options":    []string{"nosniff"},
			"Content-Security-Policy":   []string{"default-src 'self'"},
		}, Body: io.NopCloser(strings.NewReader("body"))}, nil
	})}}
	points, findings, err := suite.checkHeaders(context.Background(), "example.com")
	if err != nil || points != 45 || len(findings) != 0 {
		t.Fatalf("checkHeaders() = %d, %+v, %v", points, findings, err)
	}
}

func TestRedirectScoringRejects304(t *testing.T) {
	suite := &Suite{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotModified, Header: http.Header{"Location": []string{"https://example.com/"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}}
	points, findings, err := suite.checkRedirect(context.Background(), "example.com")
	if err != nil || points != 0 || len(findings) != 1 {
		t.Fatalf("checkRedirect() = %d, %+v, %v", points, findings, err)
	}
}

func TestValidHTTPSRedirect(t *testing.T) {
	for _, test := range []struct {
		location string
		valid    bool
	}{
		{location: "https://example.com/path", valid: true},
		{location: "http://example.com/path", valid: false},
		{location: "https://other.example/path", valid: false},
		{location: "https://user@example.com/path", valid: false},
		{location: "https://127.0.0.1/path", valid: false},
	} {
		location, err := url.Parse(test.location)
		if err != nil {
			t.Fatal(err)
		}
		if got := validHTTPSRedirect(location, "example.com"); got != test.valid {
			t.Errorf("validHTTPSRedirect(%q) = %v, want %v", test.location, got, test.valid)
		}
	}
}
