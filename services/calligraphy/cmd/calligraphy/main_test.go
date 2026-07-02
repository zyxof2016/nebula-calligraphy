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

func TestNewRouterAddsSecurityHeaders(t *testing.T) {
	router, err := newRouter(appConfig{})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Fatalf("%s = %q, want %q", header, got, want)
		}
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
		t.Fatalf("Content-Security-Policy = %q, want frame-ancestors none", got)
	}
}

func TestNewRouterLoadsConfiguredGlyphManifest(t *testing.T) {
	manifestPath := writeRuntimeGlyphManifestFixture(t, `{
	  "schema_version": "calligraphy.glyph_manifest.v1",
	  "copybook": {
	    "copybook_id": "jiuchenggong",
	    "title": "九成宫醴泉铭",
	    "style": "ou",
	    "calligrapher": "欧阳询",
	    "source_url": "https://commons.wikimedia.org/wiki/Category:%E4%B9%9D%E6%88%90%E5%AE%AE%E9%86%B4%E6%B3%89%E9%8A%98",
	    "license_status": "public_domain",
	    "attribution": "Wikimedia Commons public-domain scan"
	  },
	  "glyphs": [
	    {
	      "glyph_id": "ou-jiuchenggong-yong-original",
	      "character": "永",
	      "source_image": "s3://nebula-calligraphy/copybooks/jiuchenggong/page-001.png",
	      "crop_box": {"x": 100, "y": 120, "width": 80, "height": 96, "unit": "px"},
	      "license_status": "public_domain",
	      "review_status": "published"
	    }
	  ]
	}`)
	router, err := newRouter(appConfig{GlyphManifestFile: manifestPath})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/glyphs/search?character=永&style=ou&copybook_id=jiuchenggong", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ou-jiuchenggong-yong-original") {
		t.Fatalf("search body = %s, want manifest glyph", rec.Body.String())
	}
}

func TestNewRouterRejectsInvalidGlyphManifest(t *testing.T) {
	manifestPath := writeRuntimeGlyphManifestFixture(t, `{"schema_version":"bad"}`)

	_, err := newRouter(appConfig{GlyphManifestFile: manifestPath})
	if err == nil {
		t.Fatal("newRouter(invalid glyph manifest) error = nil, want error")
	}
}

func TestTrialRuntimeAllowsFlutterDevCorsOrigin(t *testing.T) {
	router, err := newRouter(appConfig{})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/calligraphy/glyphs/search", nil)
	req.Header.Set("Origin", "http://localhost:8088")
	req.Header.Set("Access-Control-Request-Method", "GET")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8088" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want localhost dev origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Authorization", got)
	}
}

func TestManagedRuntimeRequiresExplicitCorsOrigin(t *testing.T) {
	cfg := appConfig{
		RuntimeProfile:         "managed",
		DatabaseURL:            "postgres://calligraphy@example/calligraphy",
		IdentityIssuer:         "https://identity.example",
		IdentityBaseURL:        "https://identity.example",
		IdentityJWKSURL:        "https://identity.example/.well-known/jwks.json",
		ObjectStorageEndpoint:  "https://s3.example",
		ObjectStorageBucket:    "calligraphy-prod",
		ObjectStorageRegion:    "us-east-1",
		ObjectStorageAccessKey: "access",
		ObjectStorageSecretKey: "secret",
		AuditSink:              "https://audit.example/events",
		WebDir:                 fixtureWebDir(t),
	}
	router, err := newRouter(cfg)
	if err != nil {
		t.Fatalf("newRouter(managed) error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:8088")
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty without explicit origin", got)
	}

	cfg.AllowedOrigins = "https://calligraphy.example"
	router, err = newRouter(cfg)
	if err != nil {
		t.Fatalf("newRouter(managed with origin) error = %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://calligraphy.example")
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://calligraphy.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want configured origin", got)
	}
}

func TestManagedRuntimeConfigExposesOnlyBrowserSafeAuthSettings(t *testing.T) {
	router, err := newRouter(appConfig{
		RuntimeProfile:         "managed",
		DatabaseURL:            "postgres://calligraphy@example/calligraphy",
		AuthMode:               "oidc-pkce",
		IdentityIssuer:         "https://identity.example",
		IdentityBaseURL:        "https://identity.example",
		IdentityClientID:       "nebula-calligraphy-web",
		IdentityJWKSURL:        "https://identity.example/.well-known/jwks.json",
		IdentityHS256Secret:    "server-secret",
		ObjectStorageEndpoint:  "https://s3.example",
		ObjectStorageBucket:    "calligraphy-prod",
		ObjectStorageRegion:    "us-east-1",
		ObjectStorageAccessKey: "access",
		ObjectStorageSecretKey: "secret",
		AuditSink:              "https://audit.example/events",
		AuditToken:             "audit-token",
		WebDir:                 fixtureWebDir(t),
	})
	if err != nil {
		t.Fatalf("newRouter(managed configured) error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/runtime-config", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/runtime-config status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`"runtime_profile":"managed"`,
		`"auth_mode":"oidc-pkce"`,
		`"identity_base_url":"https://identity.example"`,
		`"identity_client_id":"nebula-calligraphy-web"`,
		`"identity_authorization_endpoint":"https://identity.example/api/v1/auth/authorize"`,
		`"identity_token_endpoint":"https://identity.example/api/v1/auth/token"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/runtime-config body = %s, want %s", body, want)
		}
	}
	for _, forbidden := range []string{"server-secret", "audit-token", "secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("/runtime-config body leaks %q: %s", forbidden, body)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self' https://identity.example") {
		t.Fatalf("Content-Security-Policy = %q, want identity origin in connect-src", csp)
	}
}

func TestProductionProfileRequiresPersistentConfig(t *testing.T) {
	_, err := newRouter(appConfig{RuntimeProfile: "production"})
	if err == nil {
		t.Fatal("newRouter(production without persistence) error = nil, want error")
	}

	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(webDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}
	router, err := newRouter(appConfig{
		RuntimeProfile: "production",
		AuthFile:       filepath.Join(dir, "auth.json"),
		DataFile:       filepath.Join(dir, "drafts.json"),
		LearningFile:   filepath.Join(dir, "learning.json"),
		AuditFile:      filepath.Join(dir, "audit.jsonl"),
		ExportDir:      filepath.Join(dir, "artifacts"),
		WebDir:         webDir,
	})
	if err != nil {
		t.Fatalf("newRouter(production configured) error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/ready status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestManagedFoundationProfileRequiresExternalServices(t *testing.T) {
	_, err := newRouter(appConfig{RuntimeProfile: "managed"})
	if err == nil {
		t.Fatal("newRouter(managed without external services) error = nil, want error")
	}

	_, err = newRouter(appConfig{
		RuntimeProfile:         "managed",
		DatabaseURL:            "postgres://calligraphy@example/calligraphy",
		IdentityIssuer:         "nebula",
		IdentityBaseURL:        "https://identity.example",
		ObjectStorageEndpoint:  "https://s3.example",
		ObjectStorageBucket:    "calligraphy-prod",
		ObjectStorageRegion:    "us-east-1",
		ObjectStorageAccessKey: "access",
		ObjectStorageSecretKey: "secret",
		AuditSink:              "https://audit.example/events",
		WebDir:                 fixtureWebDir(t),
	})
	if err == nil || !strings.Contains(err.Error(), "CALLIGRAPHY_IDENTITY_JWKS_URL or CALLIGRAPHY_IDENTITY_HS256_SECRET") {
		t.Fatalf("newRouter(managed without identity verifier config) error = %v, want identity verifier config error", err)
	}

	router, err := newRouter(appConfig{
		RuntimeProfile:         "managed",
		DatabaseURL:            "postgres://calligraphy@example/calligraphy",
		IdentityIssuer:         "https://identity.example",
		IdentityBaseURL:        "https://identity.example",
		IdentityJWKSURL:        "https://identity.example/.well-known/jwks.json",
		ObjectStorageEndpoint:  "https://s3.example",
		ObjectStorageBucket:    "calligraphy-prod",
		ObjectStorageRegion:    "us-east-1",
		ObjectStorageAccessKey: "access",
		ObjectStorageSecretKey: "secret",
		AuditSink:              "https://audit.example/events",
		WebDir:                 fixtureWebDir(t),
	})
	if err != nil {
		t.Fatalf("newRouter(managed configured) error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/ready managed status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"foundation_mode":"managed"`) {
		t.Fatalf("/ready body = %s, want managed foundation mode", rec.Body.String())
	}
}

func TestMetricsEndpointExposesRuntimeCounters(t *testing.T) {
	router, err := newRouter(appConfig{})
	if err != nil {
		t.Fatalf("newRouter() error = %v", err)
	}

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "calligraphy_http_requests_total") {
		t.Fatalf("/metrics body = %q, want request counter", rec.Body.String())
	}
}

func fixtureWebDir(t *testing.T) string {
	t.Helper()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}
	return webDir
}

func writeRuntimeGlyphManifestFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "glyph-manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(glyph manifest) error = %v", err)
	}
	return path
}
