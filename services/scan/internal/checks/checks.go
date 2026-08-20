package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"securecloud/scan/internal/store"
)

const (
	tlsPoints      = 40
	headerPoints   = 15
	redirectPoints = 15
)

type Report struct {
	Score    int
	Findings []store.Finding
}

type Suite struct {
	dialer          SecureDialer
	httpsClient     *http.Client
	httpClient      *http.Client
	now             func() time.Time
	expiryThreshold time.Duration
}

func NewSuite(timeout, expiryThreshold time.Duration) *Suite {
	dialer := SecureDialer{Dialer: &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	noRedirect := func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Suite{
		dialer:          dialer,
		httpsClient:     &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: noRedirect},
		httpClient:      &http.Client{Transport: transport.Clone(), Timeout: timeout, CheckRedirect: noRedirect},
		now:             time.Now,
		expiryThreshold: expiryThreshold,
	}
}

func (s *Suite) Run(ctx context.Context, target string) (Report, error) {
	canonical, err := NormalizeTarget(target)
	if err != nil {
		return Report{}, err
	}
	report := Report{Findings: make([]store.Finding, 0)}
	operational := 0

	tlsScore, findings, err := s.checkTLS(ctx, canonical)
	if err == nil {
		operational++
	}
	report.Score += tlsScore
	report.Findings = append(report.Findings, findings...)

	headerScore, findings, err := s.checkHeaders(ctx, canonical)
	if err == nil {
		operational++
	}
	report.Score += headerScore
	report.Findings = append(report.Findings, findings...)

	redirectScore, findings, err := s.checkRedirect(ctx, canonical)
	if err == nil {
		operational++
	}
	report.Score += redirectScore
	report.Findings = append(report.Findings, findings...)

	if operational == 0 {
		return Report{}, fmt.Errorf("all security checks failed to reach %s", canonical)
	}
	if report.Score < 0 {
		report.Score = 0
	}
	if report.Score > 100 {
		report.Score = 100
	}
	return report, nil
}

func (s *Suite) checkTLS(ctx context.Context, target string) (int, []store.Finding, error) {
	raw, err := s.dialer.DialContext(ctx, "tcp", net.JoinHostPort(target, "443"))
	if err != nil {
		return 0, []store.Finding{{Check: "tls", Severity: "high", Message: "TLS connection failed"}}, err
	}
	defer raw.Close()
	connection := tls.Client(raw, &tls.Config{ServerName: target, MinVersion: tls.VersionTLS12})
	if err := connection.HandshakeContext(ctx); err != nil {
		return 0, []store.Finding{{Check: "tls", Severity: "high", Message: "TLS certificate validation failed"}}, err
	}
	state := connection.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return 0, []store.Finding{{Check: "tls", Severity: "high", Message: "Server supplied no certificate"}}, fmt.Errorf("no peer certificate")
	}
	certificate := state.PeerCertificates[0]
	now := s.now()
	if now.Before(certificate.NotBefore) || !now.Before(certificate.NotAfter) {
		return 0, []store.Finding{{Check: "tls", Severity: "high", Message: "TLS certificate is outside its validity period"}}, nil
	}
	if certificate.NotAfter.Sub(now) <= s.expiryThreshold {
		return tlsPoints, []store.Finding{{Check: "tls_expiry", Severity: "medium", Message: fmt.Sprintf("TLS certificate expires on %s", certificate.NotAfter.UTC().Format(time.RFC3339))}}, nil
	}
	return tlsPoints, nil, nil
}

func (s *Suite) checkHeaders(ctx context.Context, target string) (int, []store.Finding, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+target+"/", nil)
	if err != nil {
		return 0, nil, err
	}
	response, err := s.httpsClient.Do(request)
	if err != nil {
		return 0, []store.Finding{{Check: "security_headers", Severity: "medium", Message: "Could not inspect HTTPS response headers"}}, err
	}
	defer response.Body.Close()
	_, _ = io.CopyN(io.Discard, response.Body, 4096)

	score := 0
	findings := make([]store.Finding, 0, 3)
	if validHSTS(response.Header.Get("Strict-Transport-Security")) {
		score += headerPoints
	} else {
		findings = append(findings, store.Finding{Check: "strict_transport_security", Severity: "medium", Message: "Strict-Transport-Security is missing or invalid"})
	}
	if strings.Contains(strings.ToLower(response.Header.Get("X-Content-Type-Options")), "nosniff") {
		score += headerPoints
	} else {
		findings = append(findings, store.Finding{Check: "x_content_type_options", Severity: "medium", Message: "X-Content-Type-Options: nosniff is missing"})
	}
	if strings.TrimSpace(response.Header.Get("Content-Security-Policy")) != "" {
		score += headerPoints
	} else {
		findings = append(findings, store.Finding{Check: "content_security_policy", Severity: "medium", Message: "Content-Security-Policy is missing"})
	}
	return score, findings, nil
}

func validHSTS(value string) bool {
	for _, directive := range strings.Split(strings.ToLower(value), ";") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") && strings.TrimPrefix(directive, "max-age=") != "0" {
			return true
		}
	}
	return false
}

func (s *Suite) checkRedirect(ctx context.Context, target string) (int, []store.Finding, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+target+"/", nil)
	if err != nil {
		return 0, nil, err
	}
	response, err := s.httpClient.Do(request)
	if err != nil {
		return 0, []store.Finding{{Check: "https_redirect", Severity: "medium", Message: "Could not inspect HTTP redirect"}}, err
	}
	defer response.Body.Close()
	_, _ = io.CopyN(io.Discard, response.Body, 4096)
	location, locationErr := response.Location()
	if response.StatusCode >= 300 && response.StatusCode < 400 && locationErr == nil && validHTTPSRedirect(location, target) {
		return redirectPoints, nil, nil
	}
	return 0, []store.Finding{{Check: "https_redirect", Severity: "medium", Message: "HTTP does not redirect directly to HTTPS on the same host"}}, nil
}

func validHTTPSRedirect(location *url.URL, target string) bool {
	if !strings.EqualFold(location.Scheme, "https") || location.User != nil {
		return false
	}
	host, err := NormalizeTarget(location.Hostname())
	return err == nil && host == target
}
