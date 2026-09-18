package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/juniorqazaq/Agro/backend/internal/analyzer"
	"github.com/juniorqazaq/Agro/backend/internal/knowledgebase"
)

func TestAnalyzeKnownDisease(t *testing.T) {
	handler := testHandler(t, analyzer.Mock{DiseaseID: "tomato_early_blight"})
	request := imageRequest(t, []byte("\x89PNG\r\n\x1a\nimage"), "leaf.png")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Disease *struct {
			ID string `json:"id"`
		} `json:"disease"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success || body.Disease == nil || body.Disease.ID != "tomato_early_blight" {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestAnalyzeUnknownDiseaseReturnsNull(t *testing.T) {
	handler := testHandler(t, analyzer.Mock{DiseaseID: "invented_id"})
	request := imageRequest(t, []byte("\x89PNG\r\n\x1a\nimage"), "leaf.png")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body struct {
		Success bool `json:"success"`
		Disease any  `json:"disease"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success || body.Disease != nil {
		t.Fatalf("expected a successful null result: %s", response.Body.String())
	}
}

func TestAnalyzeRejectsMissingFile(t *testing.T) {
	handler := testHandler(t, analyzer.Mock{DiseaseID: "tomato_early_blight"})
	request := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	request.Header.Set("Content-Type", "multipart/form-data")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestAnalyzeRejectsNonImage(t *testing.T) {
	handler := testHandler(t, analyzer.Mock{DiseaseID: "tomato_early_blight"})
	request := imageRequest(t, []byte("not an image"), "leaf.txt")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status 415, got %d", response.Code)
	}
}

func imageRequest(t *testing.T, contents []byte, filename string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/analyze", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func testHandler(t *testing.T, imageAnalyzer analyzer.Analyzer) http.Handler {
	t.Helper()
	path := filepath.Join(t.TempDir(), "diseases.json")
	data := `[{
		"id":"tomato_early_blight",
		"plant":"Tomato",
		"disease_name":"Early Blight",
		"disease_name_ru":"Альтернариоз",
		"scientific_name":"Alternaria solani",
		"symptoms":["spots"],
		"treatment":["remove leaves"],
		"reference_image_url":"https://example.com",
		"source":"test"
	}]`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := knowledgebase.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return New(imageAnalyzer, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
