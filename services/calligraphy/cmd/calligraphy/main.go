package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/handler"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

func main() {
	cfg := loadConfig()
	router, err := newRouter(cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("calligraphy service listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type appConfig struct {
	Port                   string
	RuntimeProfile         string
	AuthFile               string
	DataFile               string
	LearningFile           string
	AuditFile              string
	ExportDir              string
	WebDir                 string
	GlyphManifestFile      string
	RenderFontFile         string
	RenderCacheDir         string
	LearningTimezone       string
	SessionTTL             string
	DatabaseURL            string
	AuthMode               string
	IdentityIssuer         string
	IdentityAudience       string
	IdentityBaseURL        string
	IdentityClientID       string
	IdentityTenant         string
	IdentityAuthorizeURL   string
	IdentityTokenURL       string
	IdentityJWKSURL        string
	ObjectStorageEndpoint  string
	ObjectStorageBucket    string
	ObjectStorageRegion    string
	ObjectStorageAccessKey string
	ObjectStorageSecretKey string
	ObjectStorageToken     string
	AuditSink              string
	AuditHealthURL         string
	AuditToken             string
	AllowedOrigins         string
}

type runtimeMetrics struct {
	startedAt     time.Time
	requestsTotal atomic.Uint64
}

type readinessDependency struct {
	name  string
	check func(context.Context) error
}

func loadConfig() appConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	return appConfig{
		Port:                   port,
		RuntimeProfile:         os.Getenv("CALLIGRAPHY_RUNTIME_PROFILE"),
		AuthFile:               os.Getenv("CALLIGRAPHY_AUTH_FILE"),
		DataFile:               os.Getenv("CALLIGRAPHY_DATA_FILE"),
		LearningFile:           os.Getenv("CALLIGRAPHY_LEARNING_FILE"),
		AuditFile:              os.Getenv("CALLIGRAPHY_AUDIT_FILE"),
		ExportDir:              os.Getenv("CALLIGRAPHY_EXPORT_DIR"),
		WebDir:                 os.Getenv("CALLIGRAPHY_WEB_DIR"),
		GlyphManifestFile:      os.Getenv("CALLIGRAPHY_GLYPH_MANIFEST_FILE"),
		RenderFontFile:         os.Getenv("CALLIGRAPHY_RENDER_FONT_FILE"),
		RenderCacheDir:         os.Getenv("CALLIGRAPHY_RENDER_CACHE_DIR"),
		LearningTimezone:       os.Getenv("CALLIGRAPHY_LEARNING_TIMEZONE"),
		SessionTTL:             os.Getenv("CALLIGRAPHY_SESSION_TTL"),
		DatabaseURL:            os.Getenv("CALLIGRAPHY_DATABASE_URL"),
		AuthMode:               os.Getenv("CALLIGRAPHY_AUTH_MODE"),
		IdentityIssuer:         os.Getenv("CALLIGRAPHY_IDENTITY_ISSUER"),
		IdentityAudience:       os.Getenv("CALLIGRAPHY_IDENTITY_AUDIENCE"),
		IdentityBaseURL:        os.Getenv("CALLIGRAPHY_IDENTITY_BASE_URL"),
		IdentityClientID:       os.Getenv("CALLIGRAPHY_IDENTITY_CLIENT_ID"),
		IdentityTenant:         os.Getenv("CALLIGRAPHY_IDENTITY_TENANT"),
		IdentityAuthorizeURL:   os.Getenv("CALLIGRAPHY_IDENTITY_AUTHORIZATION_ENDPOINT"),
		IdentityTokenURL:       os.Getenv("CALLIGRAPHY_IDENTITY_TOKEN_ENDPOINT"),
		IdentityJWKSURL:        os.Getenv("CALLIGRAPHY_IDENTITY_JWKS_URL"),
		ObjectStorageEndpoint:  os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT"),
		ObjectStorageBucket:    os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_BUCKET"),
		ObjectStorageRegion:    os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_REGION"),
		ObjectStorageAccessKey: os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY"),
		ObjectStorageSecretKey: os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY"),
		ObjectStorageToken:     os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_SESSION_TOKEN"),
		AuditSink:              os.Getenv("CALLIGRAPHY_AUDIT_SINK"),
		AuditHealthURL:         os.Getenv("CALLIGRAPHY_AUDIT_HEALTH_URL"),
		AuditToken:             os.Getenv("CALLIGRAPHY_AUDIT_TOKEN"),
		AllowedOrigins:         os.Getenv("CALLIGRAPHY_ALLOWED_ORIGINS"),
	}
}

func newRouter(cfg appConfig) (http.Handler, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	router := chi.NewRouter()
	metrics := &runtimeMetrics{startedAt: time.Now()}
	router.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer, securityHeaders(cfg), corsMiddleware(cfg), metricsMiddleware(metrics))
	router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		writeMetrics(w, metrics)
	})
	router.Get("/api/v1/calligraphy/runtime-config", func(w http.ResponseWriter, r *http.Request) {
		writeRuntimeConfig(w, cfg)
	})

	layout := service.NewLayoutEngine()
	catalog, err := newGlyphCatalog(cfg)
	if err != nil {
		return nil, err
	}
	postgresDB, err := newPostgresDB(cfg)
	if err != nil {
		return nil, err
	}
	artworkStore, err := newArtworkStore(cfg, postgresDB)
	if err != nil {
		return nil, err
	}
	learningStore, err := newLearningStore(cfg, postgresDB)
	if err != nil {
		return nil, err
	}
	learningLocation, err := newLearningLocation(cfg)
	if err != nil {
		return nil, err
	}
	authStore, err := newAuthStore(cfg, postgresDB)
	if err != nil {
		return nil, err
	}
	authService := service.NewAuthService(authStore)
	sessionTTL, err := newSessionTTL(cfg)
	if err != nil {
		return nil, err
	}
	authService.SetSessionTTL(sessionTTL)
	artifactStore := newArtifactStore(cfg)
	artworkService := service.NewArtworkService(artworkStore, layout, service.NewSVGRenderer(), artifactStore)
	artworkService.SetPNGRenderer(service.NewArtworkPNGRenderer(cfg.RenderFontFile))
	learningService := service.NewLearningService(learningStore, catalog)
	learningService.SetLearningLocation(learningLocation)
	identityVerifier := newIdentityVerifier(cfg, authService)
	auditLogger := newAuditLogger(cfg)
	calligraphyHandler := handler.New(
		catalog,
		layout,
		artworkService,
		learningService,
		authService,
		auditLogger,
		identityVerifier,
	)
	calligraphyHandler.SetGlyphRenderer(service.NewGlyphImageRenderer(cfg.RenderFontFile, cfg.RenderCacheDir))
	calligraphyHandler.SetLocalAuthEnabled(cfg.RuntimeProfile != "managed")
	handler.RegisterRoutes(router, calligraphyHandler)
	readinessDependencies := newReadinessDependencies(cfg, postgresDB, artifactStore, identityVerifier, auditLogger)
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		writeReadiness(w, r, cfg, readinessDependencies)
	})
	if cfg.WebDir != "" {
		router.Handle("/*", spaFileServer(cfg.WebDir))
	}
	if cfg.ExportDir != "" {
		router.Handle("/artifacts/*", http.StripPrefix("/artifacts/", http.FileServer(http.Dir(cfg.ExportDir))))
	}
	return router, nil
}

func spaFileServer(webDir string) http.Handler {
	files := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." {
			setSPACacheHeaders(w, "index.html")
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(filepath.Join(webDir, cleanPath))
		if err == nil && !info.IsDir() {
			setSPACacheHeaders(w, cleanPath)
			files.ServeHTTP(w, r)
			return
		}
		setSPACacheHeaders(w, "index.html")
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})
}

func setSPACacheHeaders(w http.ResponseWriter, assetPath string) {
	switch filepath.ToSlash(assetPath) {
	case "index.html", "flutter_service_worker.js", "version.json":
		w.Header().Set("Cache-Control", "no-cache")
	case "main.dart.js", "flutter_bootstrap.js", "flutter.js", "manifest.json":
		w.Header().Set("Cache-Control", "public, max-age=3600")
	default:
		if strings.HasPrefix(assetPath, "assets/") ||
			strings.HasPrefix(assetPath, "canvaskit/") ||
			strings.HasPrefix(assetPath, "icons/") {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
	}
}

func newGlyphCatalog(cfg appConfig) (service.GlyphCatalog, error) {
	fallback := service.NewInMemoryGlyphCatalog()
	if cfg.GlyphManifestFile == "" {
		return fallback, nil
	}
	fileCatalog, err := service.NewFileGlyphCatalog(cfg.GlyphManifestFile)
	if err != nil {
		return nil, fmt.Errorf("load CALLIGRAPHY_GLYPH_MANIFEST_FILE: %w", err)
	}
	if cfg.RuntimeProfile == "production" || cfg.RuntimeProfile == "managed" {
		if len(fileCatalog.Search(service.GlyphSearchParams{})) == 0 {
			return nil, errors.New("CALLIGRAPHY_GLYPH_MANIFEST_FILE must contain at least one published, unrestricted glyph")
		}
		return fileCatalog, nil
	}
	return service.NewCompositeGlyphCatalog(fileCatalog, fallback), nil
}

func newLearningLocation(cfg appConfig) (*time.Location, error) {
	timezone := strings.TrimSpace(cfg.LearningTimezone)
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid CALLIGRAPHY_LEARNING_TIMEZONE %q: %w", timezone, err)
	}
	return location, nil
}

func newSessionTTL(cfg appConfig) (time.Duration, error) {
	raw := strings.TrimSpace(cfg.SessionTTL)
	if raw == "" {
		return 24 * time.Hour, nil
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl <= 0 {
		return 0, fmt.Errorf("invalid CALLIGRAPHY_SESSION_TTL %q: must be a positive Go duration", raw)
	}
	return ttl, nil
}

func validateConfig(cfg appConfig) error {
	switch cfg.RuntimeProfile {
	case "", "trial":
		return nil
	case "production":
		return validateRequired("production profile requires persistent configuration", map[string]string{
			"CALLIGRAPHY_AUTH_FILE":           cfg.AuthFile,
			"CALLIGRAPHY_DATA_FILE":           cfg.DataFile,
			"CALLIGRAPHY_LEARNING_FILE":       cfg.LearningFile,
			"CALLIGRAPHY_AUDIT_FILE":          cfg.AuditFile,
			"CALLIGRAPHY_EXPORT_DIR":          cfg.ExportDir,
			"CALLIGRAPHY_WEB_DIR":             cfg.WebDir,
			"CALLIGRAPHY_GLYPH_MANIFEST_FILE": cfg.GlyphManifestFile,
		})
	case "managed":
		if err := validateRequired("managed profile requires external foundation configuration", map[string]string{
			"CALLIGRAPHY_DATABASE_URL":              cfg.DatabaseURL,
			"CALLIGRAPHY_IDENTITY_ISSUER":           cfg.IdentityIssuer,
			"CALLIGRAPHY_IDENTITY_AUDIENCE":         cfg.IdentityAudience,
			"CALLIGRAPHY_IDENTITY_BASE_URL":         cfg.IdentityBaseURL,
			"CALLIGRAPHY_IDENTITY_TENANT":           cfg.IdentityTenant,
			"CALLIGRAPHY_IDENTITY_JWKS_URL":         cfg.IdentityJWKSURL,
			"CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT":   cfg.ObjectStorageEndpoint,
			"CALLIGRAPHY_OBJECT_STORAGE_BUCKET":     cfg.ObjectStorageBucket,
			"CALLIGRAPHY_OBJECT_STORAGE_REGION":     cfg.ObjectStorageRegion,
			"CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY": cfg.ObjectStorageAccessKey,
			"CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY": cfg.ObjectStorageSecretKey,
			"CALLIGRAPHY_AUDIT_SINK":                cfg.AuditSink,
			"CALLIGRAPHY_AUDIT_HEALTH_URL":          cfg.AuditHealthURL,
			"CALLIGRAPHY_WEB_DIR":                   cfg.WebDir,
			"CALLIGRAPHY_GLYPH_MANIFEST_FILE":       cfg.GlyphManifestFile,
		}); err != nil {
			return err
		}
		if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,49}$`).MatchString(cfg.IdentityTenant) {
			return errors.New("CALLIGRAPHY_IDENTITY_TENANT must be a lowercase tenant slug")
		}
		if !strings.HasPrefix(cfg.AuditSink, "http://") && !strings.HasPrefix(cfg.AuditSink, "https://") {
			return errors.New("managed profile requires CALLIGRAPHY_AUDIT_SINK to be an http or https URL")
		}
		if !strings.HasPrefix(cfg.AuditHealthURL, "http://") && !strings.HasPrefix(cfg.AuditHealthURL, "https://") {
			return errors.New("managed profile requires CALLIGRAPHY_AUDIT_HEALTH_URL to be an http or https URL")
		}
		switch runtimeAuthMode(cfg) {
		case "oidc-pkce":
			if err := validateRequired("managed oidc-pkce auth requires browser OIDC settings", map[string]string{
				"CALLIGRAPHY_IDENTITY_CLIENT_ID":              cfg.IdentityClientID,
				"CALLIGRAPHY_IDENTITY_AUTHORIZATION_ENDPOINT": identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/authorize", cfg.IdentityAuthorizeURL),
				"CALLIGRAPHY_IDENTITY_TOKEN_ENDPOINT":         identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/token", cfg.IdentityTokenURL),
			}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported CALLIGRAPHY_AUTH_MODE %q", runtimeAuthMode(cfg))
		}
		return nil
	default:
		return fmt.Errorf("unsupported CALLIGRAPHY_RUNTIME_PROFILE %q", cfg.RuntimeProfile)
	}
}

func validateRequired(prefix string, values map[string]string) error {
	missing := make([]string, 0)
	for name, value := range values {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s: %s", prefix, strings.Join(missing, ", "))
	}
	return nil
}

func writeReadiness(w http.ResponseWriter, r *http.Request, cfg appConfig, dependencies []readinessDependency) {
	w.Header().Set("Content-Type", "application/json")
	if err := validateConfig(cfg); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "not_ready", "message": err.Error()})
		return
	}
	foundationMode := "local"
	if cfg.RuntimeProfile == "managed" {
		foundationMode = "managed"
	}
	componentStatus := make(map[string]string, len(dependencies))
	for _, dependency := range dependencies {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := dependency.check(ctx)
		cancel()
		if err != nil {
			componentStatus[dependency.name] = "unavailable"
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":          "not_ready",
				"service":         "calligraphy",
				"foundation_mode": foundationMode,
				"components":      componentStatus,
				"message":         dependency.name + " is unavailable",
			})
			return
		}
		componentStatus[dependency.name] = "ready"
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":          "ready",
		"service":         "calligraphy",
		"foundation_mode": foundationMode,
		"components":      componentStatus,
	})
}

func newReadinessDependencies(cfg appConfig, db *sql.DB, artifacts service.ArtifactStore, identity service.IdentityVerifier, audit service.AuditLogger) []readinessDependency {
	dependencies := make([]readinessDependency, 0, 5)
	if cfg.RuntimeProfile == "production" {
		dependencies = append(dependencies, readinessDependency{
			name: "persistent_storage",
			check: func(context.Context) error {
				return checkLocalPersistentStorage(cfg)
			},
		})
	}
	if cfg.RuntimeProfile == "managed" {
		dependencies = append(dependencies, readinessDependency{
			name: "postgres",
			check: func(ctx context.Context) error {
				var migrated bool
				err := db.QueryRowContext(ctx, `
				SELECT to_regclass('public.calligraphy_schema_migrations') IS NOT NULL
				  AND to_regclass('public.calligraphy_auth_users') IS NOT NULL
				  AND to_regclass('public.calligraphy_auth_sessions') IS NOT NULL
				  AND to_regclass('public.calligraphy_artwork_drafts') IS NOT NULL
				  AND to_regclass('public.calligraphy_learning_favorites') IS NOT NULL
				  AND to_regclass('public.calligraphy_learning_practice') IS NOT NULL
				  AND EXISTS (
					SELECT 1
					FROM calligraphy_schema_migrations
					WHERE version = 2
				  )
			`).Scan(&migrated)
				if err != nil {
					return err
				}
				if !migrated {
					return errors.New("calligraphy database migration is incomplete")
				}
				return nil
			},
		})
	}
	if checker, ok := artifacts.(interface{ Check(context.Context) error }); ok {
		dependencies = append(dependencies, readinessDependency{name: "object_storage", check: checker.Check})
	}
	if checker, ok := identity.(interface{ Check(context.Context) error }); ok {
		dependencies = append(dependencies, readinessDependency{name: "identity", check: checker.Check})
	}
	if checker, ok := audit.(interface{ Check(context.Context) error }); ok {
		dependencies = append(dependencies, readinessDependency{name: "audit", check: checker.Check})
	}
	if cfg.RuntimeProfile == "production" || cfg.RuntimeProfile == "managed" {
		dependencies = append(dependencies, readinessDependency{
			name: "web_assets",
			check: func(context.Context) error {
				entrypoint := filepath.Join(cfg.WebDir, "index.html")
				info, err := os.Stat(entrypoint)
				if err != nil {
					return fmt.Errorf("read web entrypoint: %w", err)
				}
				if !info.Mode().IsRegular() {
					return errors.New("web entrypoint is not a regular file")
				}
				file, err := os.Open(entrypoint)
				if err != nil {
					return fmt.Errorf("open web entrypoint: %w", err)
				}
				if err := file.Close(); err != nil {
					return fmt.Errorf("close web entrypoint: %w", err)
				}
				return nil
			},
		})
	}
	return dependencies
}

func checkLocalPersistentStorage(cfg appConfig) error {
	directories := []string{
		filepath.Dir(cfg.AuthFile),
		filepath.Dir(cfg.DataFile),
		filepath.Dir(cfg.LearningFile),
		filepath.Dir(cfg.AuditFile),
		cfg.ExportDir,
	}
	if cfg.RenderCacheDir != "" {
		directories = append(directories, cfg.RenderCacheDir)
	}
	checked := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		cleaned := filepath.Clean(directory)
		if _, ok := checked[cleaned]; ok {
			continue
		}
		checked[cleaned] = struct{}{}
		if err := checkWritableDirectory(cleaned); err != nil {
			return err
		}
	}
	if info, err := os.Stat(cfg.AuditFile); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("audit path is not a regular file")
		}
		file, openErr := os.OpenFile(cfg.AuditFile, os.O_WRONLY|os.O_APPEND, 0)
		if openErr != nil {
			return fmt.Errorf("open audit file for append: %w", openErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("close audit file: %w", closeErr)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect audit file: %w", err)
	}
	return nil
}

func checkWritableDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create persistent directory %s: %w", directory, err)
	}
	probe, err := os.CreateTemp(directory, ".calligraphy-readiness-*")
	if err != nil {
		return fmt.Errorf("create persistence probe in %s: %w", directory, err)
	}
	probePath := probe.Name()
	if _, err := probe.Write([]byte{1}); err != nil {
		_ = probe.Close()
		_ = os.Remove(probePath)
		return fmt.Errorf("write persistence probe in %s: %w", directory, err)
	}
	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return fmt.Errorf("close persistence probe in %s: %w", directory, err)
	}
	if err := os.Remove(probePath); err != nil {
		return fmt.Errorf("remove persistence probe in %s: %w", directory, err)
	}
	return nil
}

type publicRuntimeConfig struct {
	RuntimeProfile                string `json:"runtime_profile"`
	AuthMode                      string `json:"auth_mode"`
	IdentityBaseURL               string `json:"identity_base_url,omitempty"`
	IdentityClientID              string `json:"identity_client_id,omitempty"`
	IdentityTenant                string `json:"identity_tenant,omitempty"`
	IdentityAuthorizationEndpoint string `json:"identity_authorization_endpoint,omitempty"`
	IdentityTokenEndpoint         string `json:"identity_token_endpoint,omitempty"`
}

func writeRuntimeConfig(w http.ResponseWriter, cfg appConfig) {
	w.Header().Set("Content-Type", "application/json")
	profile := cfg.RuntimeProfile
	if profile == "" {
		profile = "trial"
	}
	payload := publicRuntimeConfig{
		RuntimeProfile:                profile,
		AuthMode:                      runtimeAuthMode(cfg),
		IdentityBaseURL:               cfg.IdentityBaseURL,
		IdentityClientID:              cfg.IdentityClientID,
		IdentityTenant:                cfg.IdentityTenant,
		IdentityAuthorizationEndpoint: identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/authorize", cfg.IdentityAuthorizeURL),
		IdentityTokenEndpoint:         identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/token", cfg.IdentityTokenURL),
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func runtimeAuthMode(cfg appConfig) string {
	mode := strings.TrimSpace(cfg.AuthMode)
	if mode != "" {
		return mode
	}
	if cfg.RuntimeProfile == "managed" {
		return "oidc-pkce"
	}
	return "local"
}

func identityEndpoint(baseURL, defaultPath, override string) string {
	if override != "" {
		return override
	}
	if baseURL == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + defaultPath
}

func securityHeaders(cfg appConfig) func(http.Handler) http.Handler {
	connectSources := cspConnectSources(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src %s; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'", strings.Join(connectSources, " ")))
			next.ServeHTTP(w, r)
		})
	}
}

func cspConnectSources(cfg appConfig) []string {
	sources := []string{"'self'"}
	seen := map[string]bool{"'self'": true}
	for _, rawURL := range []string{
		cfg.IdentityBaseURL,
		cfg.IdentityAuthorizeURL,
		cfg.IdentityTokenURL,
	} {
		origin := urlOrigin(rawURL)
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		sources = append(sources, origin)
	}
	return sources
}

func corsMiddleware(cfg appConfig) func(http.Handler) http.Handler {
	allowed := allowedOrigins(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "false")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowedOrigins(cfg appConfig) map[string]bool {
	origins := map[string]bool{}
	if cfg.RuntimeProfile == "" || cfg.RuntimeProfile == "trial" {
		origins["http://localhost:8088"] = true
		origins["http://127.0.0.1:8088"] = true
	}
	for _, origin := range strings.Split(cfg.AllowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		origins[origin] = true
	}
	return origins
}

func urlOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func metricsMiddleware(metrics *runtimeMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics.requestsTotal.Add(1)
			next.ServeHTTP(w, r)
		})
	}
}

func writeMetrics(w http.ResponseWriter, metrics *runtimeMetrics) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	uptime := time.Since(metrics.startedAt).Seconds()
	_, _ = fmt.Fprintf(w, "# HELP calligraphy_http_requests_total Total HTTP requests handled by this process.\n")
	_, _ = fmt.Fprintf(w, "# TYPE calligraphy_http_requests_total counter\n")
	_, _ = fmt.Fprintf(w, "calligraphy_http_requests_total %d\n", metrics.requestsTotal.Load())
	_, _ = fmt.Fprintf(w, "# HELP calligraphy_process_uptime_seconds Process uptime in seconds.\n")
	_, _ = fmt.Fprintf(w, "# TYPE calligraphy_process_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "calligraphy_process_uptime_seconds %.0f\n", uptime)
}

func newPostgresDB(cfg appConfig) (*sql.DB, error) {
	if cfg.RuntimeProfile != "managed" {
		return nil, nil
	}
	return service.OpenPostgres(cfg.DatabaseURL)
}

func newArtworkStore(cfg appConfig, db *sql.DB) (service.ArtworkStore, error) {
	if cfg.RuntimeProfile == "managed" {
		return service.NewPostgresArtworkStore(db), nil
	}
	if cfg.DataFile == "" {
		return service.NewInMemoryArtworkStore(), nil
	}
	return service.NewFileArtworkStore(cfg.DataFile)
}

func newLearningStore(cfg appConfig, db *sql.DB) (service.LearningStore, error) {
	if cfg.RuntimeProfile == "managed" {
		return service.NewPostgresLearningStore(db), nil
	}
	if cfg.LearningFile == "" {
		return service.NewInMemoryLearningStore(), nil
	}
	return service.NewFileLearningStore(cfg.LearningFile)
}

func newAuthStore(cfg appConfig, db *sql.DB) (service.AuthStore, error) {
	if cfg.RuntimeProfile == "managed" {
		return service.NewPostgresAuthStore(db), nil
	}
	if cfg.AuthFile == "" {
		return service.NewInMemoryAuthStore(), nil
	}
	return service.NewFileAuthStore(cfg.AuthFile)
}

func newArtifactStore(cfg appConfig) service.ArtifactStore {
	if cfg.RuntimeProfile == "managed" {
		return service.NewS3ArtifactStore(service.S3ArtifactStoreConfig{
			Endpoint:        cfg.ObjectStorageEndpoint,
			Bucket:          cfg.ObjectStorageBucket,
			Region:          cfg.ObjectStorageRegion,
			AccessKeyID:     cfg.ObjectStorageAccessKey,
			SecretAccessKey: cfg.ObjectStorageSecretKey,
			SessionToken:    cfg.ObjectStorageToken,
		})
	}
	if cfg.ExportDir == "" {
		return nil
	}
	return service.NewLocalArtifactStore(cfg.ExportDir)
}

func newAuditLogger(cfg appConfig) service.AuditLogger {
	if cfg.RuntimeProfile == "managed" {
		return service.NewHTTPAuditLogger(service.HTTPAuditLoggerConfig{
			Endpoint:       cfg.AuditSink,
			HealthEndpoint: cfg.AuditHealthURL,
			BearerToken:    cfg.AuditToken,
		})
	}
	if cfg.AuditFile == "" {
		return service.NoopAuditLogger{}
	}
	return service.NewFileAuditLogger(cfg.AuditFile)
}

func newIdentityVerifier(cfg appConfig, fallback service.IdentityVerifier) service.IdentityVerifier {
	if cfg.RuntimeProfile != "managed" {
		return fallback
	}
	return service.NewJWKSIdentityVerifier(service.JWKSIdentityConfig{
		Issuer:   cfg.IdentityIssuer,
		Audience: cfg.IdentityAudience,
		JWKSURL:  cfg.IdentityJWKSURL,
	})
}
