package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/juniorqazaq/Agro/backend/internal/analyzer"
	"github.com/juniorqazaq/Agro/backend/internal/domain"
	"github.com/juniorqazaq/Agro/backend/internal/knowledgebase"
)

const (
	maxImageSize   = 10 << 20
	maxRequestSize = maxImageSize + (1 << 20)
)

type Handler struct {
	analyzer analyzer.Analyzer
	store    *knowledgebase.Store
	logger   *slog.Logger
}

type analyzeResponse struct {
	Success bool            `json:"success"`
	Disease *domain.Disease `json:"disease"`
	Message string          `json:"message,omitempty"`
}

type errorResponse struct {
	Success bool     `json:"success"`
	Error   apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(analyzer analyzer.Analyzer, store *knowledgebase.Store, logger *slog.Logger) http.Handler {
	h := &Handler{analyzer: analyzer, store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("POST /api/analyze", h.analyze)
	return cors(mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) analyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Send an image in multipart/form-data using the 'file' field (maximum 10 MB).")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "FILE_REQUIRED", "The 'file' image field is required.")
		return
	}
	defer file.Close()

	image, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FILE", "Could not read the uploaded image.")
		return
	}
	if len(image) == 0 {
		writeError(w, http.StatusBadRequest, "EMPTY_FILE", "The uploaded image is empty.")
		return
	}
	if len(image) > maxImageSize {
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "The image must not exceed 10 MB.")
		return
	}

	mimeType := http.DetectContentType(image)
	if !allowedImageType(mimeType) {
		writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only JPEG, PNG, or WebP images are supported.")
		return
	}

	diseaseID, err := h.analyzer.Analyze(r.Context(), image, mimeType, h.store.IDs())
	if err != nil {
		if errors.Is(err, analyzer.ErrNotRecognized) {
			writeJSON(w, http.StatusOK, analyzeResponse{
				Success: true,
				Disease: nil,
				Message: "Disease could not be recognized from this image.",
			})
			return
		}
		h.logger.Error("image analysis failed", "error", err, "filename", header.Filename)
		writeError(w, http.StatusBadGateway, "AI_UNAVAILABLE", "The image analysis service is temporarily unavailable.")
		return
	}

	disease, ok := h.store.Find(diseaseID)
	if !ok {
		h.logger.Warn("AI returned an unknown disease id", "disease_id", diseaseID)
		writeJSON(w, http.StatusOK, analyzeResponse{
			Success: true,
			Disease: nil,
			Message: "Disease could not be recognized from this image.",
		})
		return
	}

	writeJSON(w, http.StatusOK, analyzeResponse{
		Success: true,
		Disease: &disease,
	})
}

func allowedImageType(mimeType string) bool {
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{
		Success: false,
		Error: apiError{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
