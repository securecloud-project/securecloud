package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"securecloud/scan/internal/checks"
	"securecloud/scan/internal/store"
)

const maxRequestBody = 4096

type Handler struct {
	store  *store.Store
	logger *slog.Logger
}

func New(scanStore *store.Store, logger *slog.Logger) *Handler {
	return &Handler{store: scanStore, logger: logger}
}

func (h *Handler) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(h.requestLogger)
	router.Get("/healthz", h.healthz)
	router.Get("/readyz", h.readyz)
	router.Post("/scan", h.createScan)
	router.Get("/scan/{id}", h.getScan)
	router.Get("/scans", h.listScans)
	return router
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
