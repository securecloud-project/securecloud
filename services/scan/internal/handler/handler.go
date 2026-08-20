package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"securecloud/scan/internal/checks"
	"securecloud/scan/internal/store"
)

const maxRequestBody = 4096

type Handler struct {
	store     *store.Store
	logger    *slog.Logger
	submitter interface{ Enqueue(string) bool }
	verifier  interface {
		Verify(context.Context, string) error
	}
	limiter *tokenBucket
}

type Option func(*Handler)

func WithSubmitter(submitter interface{ Enqueue(string) bool }) Option {
	return func(handler *Handler) { handler.submitter = submitter }
}

func WithVerifier(verifier interface {
	Verify(context.Context, string) error
}) Option {
	return func(handler *Handler) { handler.verifier = verifier }
}

func WithRateLimit(requestsPerSecond, burst int) Option {
	return func(handler *Handler) { handler.limiter = newTokenBucket(float64(requestsPerSecond), float64(burst)) }
}

func New(scanStore *store.Store, logger *slog.Logger, options ...Option) *Handler {
	handler := &Handler{store: scanStore, logger: logger, limiter: newTokenBucket(20, 40)}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (h *Handler) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(h.requestLogger)
	router.Get("/healthz", h.healthz)
	router.Get("/readyz", h.readyz)
	router.Group(func(protected chi.Router) {
		protected.Use(h.limitRequests)
		protected.Use(h.requireAuthorization)
		protected.Post("/scan", h.createScan)
		protected.Get("/scan/{id}", h.getScan)
		protected.Get("/scans", h.listScans)
	})
	return router
}

func (h *Handler) requireAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.verifier == nil {
			next.ServeHTTP(w, r)
			return
		}
		authorization := r.Header.Get("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")) == "" {
			writeError(w, http.StatusUnauthorized, "bearer token is required")
			return
		}
		if err := h.verifier.Verify(r.Context(), authorization); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) limitRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.limiter.Allow(time.Now()) {
			w.Header().Set("Retry-After", "1")
			writeError(w, http.StatusTooManyRequests, "request rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	return &tokenBucket{rate: rate, burst: burst, tokens: burst, last: time.Now()}
}

func (b *tokenBucket) Allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		h.logger.Error("database readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type createScanRequest struct {
	Target string `json:"target"`
}

func (h *Handler) createScan(w http.ResponseWriter, r *http.Request) {
	var request createScanRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}
	canonicalTarget, err := checks.NormalizeTarget(request.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scan, err := h.store.CreateScan(r.Context(), canonicalTarget)
	if err != nil {
		h.logger.Error("failed to create scan", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create scan")
		return
	}
	if h.submitter != nil && !h.submitter.Enqueue(scan.ID) {
		_ = h.store.FailScan(r.Context(), scan.ID, "scan queue is full")
		writeError(w, http.StatusServiceUnavailable, "scan queue is full; retry later")
		return
	}
	h.logger.Info("scan created", "scan_id", scan.ID, "target", scan.Target)
	writeJSON(w, http.StatusAccepted, scan)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func (h *Handler) getScan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scan, err := h.store.GetScan(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "scan not found")
		return
	}
	if err != nil {
		h.logger.Error("failed to retrieve scan", "scan_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve scan")
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) listScans(w http.ResponseWriter, r *http.Request) {
	scans, err := h.store.ListScans(r.Context())
	if err != nil {
		h.logger.Error("failed to list scans", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list scans")
		return
	}
	writeJSON(w, http.StatusOK, scans)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (h *Handler) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		h.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration_ms", time.Since(start).Milliseconds())
	})
}
