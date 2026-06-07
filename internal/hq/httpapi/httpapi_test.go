package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteJSON(recorder, http.StatusCreated, map[string]string{"status": "ok"})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body status = %q, want ok", body["status"])
	}
}

func TestStaticHandlerServesExistingFile(t *testing.T) {
	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "asset.txt"), []byte("asset"), 0644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/asset.txt", nil)

	StaticHandler(webRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if strings.TrimSpace(recorder.Body.String()) != "asset" {
		t.Fatalf("body = %q, want asset", recorder.Body.String())
	}
}

func TestStaticHandlerFallsBackToIndex(t *testing.T) {
	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/student/pet/sunny-town", nil)

	StaticHandler(webRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if strings.TrimSpace(recorder.Body.String()) != "index" {
		t.Fatalf("body = %q, want index", recorder.Body.String())
	}
}

func TestStaticHandlerRejectsUnsupportedMethods(t *testing.T) {
	webRoot := t.TempDir()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	StaticHandler(webRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestLogRequestsCallsNext(t *testing.T) {
	called := false
	handler := LogRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("next handler was not called")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}
