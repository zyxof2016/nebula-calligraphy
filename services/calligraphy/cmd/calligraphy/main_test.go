package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRouterServesConfiguredWebApp(t *testing.T) {
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html><title>Nebula Calligraphy</title></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}

	router, err := newRouter(appConfig{WebDir: webDir})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Nebula Calligraphy") {
		t.Fatalf("body = %q, want web app html", rec.Body.String())
	}
}

func TestNewRouterServesConfiguredArtifacts(t *testing.T) {
	artifactDir := t.TempDir()
	artifactPath := filepath.Join(artifactDir, "artwork-000001", "export-000001.svg")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte(`<svg id="export"></svg>`), 0o644); err != nil {
		t.Fatalf("WriteFile(export) error = %v", err)
	}

	router, err := newRouter(appConfig{ExportDir: artifactDir})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/artifacts/artwork-000001/export-000001.svg", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<svg id="export"></svg>`) {
		t.Fatalf("body = %q, want artifact content", rec.Body.String())
	}
}
