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
	"strings"
	"sync/atomic"
	"syscall"
	"time"

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
	DatabaseURL            string
	AuthMode               string
	IdentityIssuer         string
	IdentityBaseURL        string
	IdentityClientID       string
	IdentityAuthorizeURL   string
	IdentityTokenURL       string
	IdentityLoginURL       string
	IdentityJWKSURL        string
	IdentityHS256Secret    string
	ObjectStorageEndpoint  string
	ObjectStorageBucket    string
	ObjectStorageRegion    string
	ObjectStorageAccessKey string
	ObjectStorageSecretKey string
	AuditSink              string
	AuditToken             string
	AllowedOrigins         string
}

type runtimeMetrics struct {
	startedAt     time.Time
	requestsTotal atomic.Uint64
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
		DatabaseURL:            os.Getenv("CALLIGRAPHY_DATABASE_URL"),
		AuthMode:               os.Getenv("CALLIGRAPHY_AUTH_MODE"),
		IdentityIssuer:         os.Getenv("CALLIGRAPHY_IDENTITY_ISSUER"),
		IdentityBaseURL:        os.Getenv("CALLIGRAPHY_IDENTITY_BASE_URL"),
		IdentityClientID:       os.Getenv("CALLIGRAPHY_IDENTITY_CLIENT_ID"),
		IdentityAuthorizeURL:   os.Getenv("CALLIGRAPHY_IDENTITY_AUTHORIZATION_ENDPOINT"),
		IdentityTokenURL:       os.Getenv("CALLIGRAPHY_IDENTITY_TOKEN_ENDPOINT"),
		IdentityLoginURL:       os.Getenv("CALLIGRAPHY_IDENTITY_LOGIN_ENDPOINT"),
		IdentityJWKSURL:        os.Getenv("CALLIGRAPHY_IDENTITY_JWKS_URL"),
		IdentityHS256Secret:    os.Getenv("CALLIGRAPHY_IDENTITY_HS256_SECRET"),
		ObjectStorageEndpoint:  os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT"),
		ObjectStorageBucket:    os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_BUCKET"),
		ObjectStorageRegion:    os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_REGION"),
		ObjectStorageAccessKey: os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY"),
		ObjectStorageSecretKey: os.Getenv("CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY"),
		AuditSink:              os.Getenv("CALLIGRAPHY_AUDIT_SINK"),
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
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		writeReadiness(w, cfg)
	})
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
	authStore, err := newAuthStore(cfg, postgresDB)
	if err != nil {
		return nil, err
	}
	authService := service.NewAuthService(authStore)
	handler.RegisterRoutes(router, handler.New(
		catalog,
		layout,
		service.NewArtworkService(artworkStore, layout, service.NewSVGRenderer(), newArtifactStore(cfg)),
		service.NewLearningService(learningStore, catalog),
		authService,
		newAuditLogger(cfg),
		newIdentityVerifier(cfg, authService),
	))
	if cfg.WebDir != "" {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, cfg.WebDir+"/index.html")
		})
		router.Handle("/app/*", http.StripPrefix("/app/", http.FileServer(http.Dir(cfg.WebDir))))
	}
	if cfg.ExportDir != "" {
		router.Handle("/artifacts/*", http.StripPrefix("/artifacts/", http.FileServer(http.Dir(cfg.ExportDir))))
	}
	return router, nil
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
	return service.NewCompositeGlyphCatalog(fileCatalog, fallback), nil
}

func validateConfig(cfg appConfig) error {
	switch cfg.RuntimeProfile {
	case "", "trial":
		return nil
	case "production":
		return validateRequired("production profile requires persistent configuration", map[string]string{
			"CALLIGRAPHY_AUTH_FILE":     cfg.AuthFile,
			"CALLIGRAPHY_DATA_FILE":     cfg.DataFile,
			"CALLIGRAPHY_LEARNING_FILE": cfg.LearningFile,
			"CALLIGRAPHY_AUDIT_FILE":    cfg.AuditFile,
			"CALLIGRAPHY_EXPORT_DIR":    cfg.ExportDir,
			"CALLIGRAPHY_WEB_DIR":       cfg.WebDir,
		})
	case "managed":
		if err := validateRequired("managed profile requires external foundation configuration", map[string]string{
			"CALLIGRAPHY_DATABASE_URL":              cfg.DatabaseURL,
			"CALLIGRAPHY_IDENTITY_ISSUER":           cfg.IdentityIssuer,
			"CALLIGRAPHY_IDENTITY_BASE_URL":         cfg.IdentityBaseURL,
			"CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT":   cfg.ObjectStorageEndpoint,
			"CALLIGRAPHY_OBJECT_STORAGE_BUCKET":     cfg.ObjectStorageBucket,
			"CALLIGRAPHY_OBJECT_STORAGE_REGION":     cfg.ObjectStorageRegion,
			"CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY": cfg.ObjectStorageAccessKey,
			"CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY": cfg.ObjectStorageSecretKey,
			"CALLIGRAPHY_AUDIT_SINK":                cfg.AuditSink,
			"CALLIGRAPHY_WEB_DIR":                   cfg.WebDir,
		}); err != nil {
			return err
		}
		if cfg.IdentityJWKSURL == "" && cfg.IdentityHS256Secret == "" {
			return errors.New("managed profile requires CALLIGRAPHY_IDENTITY_JWKS_URL or CALLIGRAPHY_IDENTITY_HS256_SECRET")
		}
		if !strings.HasPrefix(cfg.AuditSink, "http://") && !strings.HasPrefix(cfg.AuditSink, "https://") {
			return errors.New("managed profile requires CALLIGRAPHY_AUDIT_SINK to be an http or https URL")
		}
		switch runtimeAuthMode(cfg) {
		case "nebula-direct":
			if identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/login", cfg.IdentityLoginURL) == "" {
				return errors.New("managed nebula-direct auth requires CALLIGRAPHY_IDENTITY_BASE_URL or CALLIGRAPHY_IDENTITY_LOGIN_ENDPOINT")
			}
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

func writeReadiness(w http.ResponseWriter, cfg appConfig) {
	w.Header().Set("Content-Type", "application/json")
	if err := validateConfig(cfg); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"not_ready","message":%q}`, err.Error())))
		return
	}
	w.WriteHeader(http.StatusOK)
	foundationMode := "local"
	if cfg.RuntimeProfile == "managed" {
		foundationMode = "managed"
	}
	_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ready","service":"calligraphy","foundation_mode":%q}`, foundationMode)))
}

type publicRuntimeConfig struct {
	RuntimeProfile                string `json:"runtime_profile"`
	AuthMode                      string `json:"auth_mode"`
	IdentityBaseURL               string `json:"identity_base_url,omitempty"`
	IdentityClientID              string `json:"identity_client_id,omitempty"`
	IdentityAuthorizationEndpoint string `json:"identity_authorization_endpoint,omitempty"`
	IdentityTokenEndpoint         string `json:"identity_token_endpoint,omitempty"`
	IdentityLoginEndpoint         string `json:"identity_login_endpoint,omitempty"`
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
		IdentityAuthorizationEndpoint: identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/authorize", cfg.IdentityAuthorizeURL),
		IdentityTokenEndpoint:         identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/token", cfg.IdentityTokenURL),
		IdentityLoginEndpoint:         identityEndpoint(cfg.IdentityBaseURL, "/api/v1/auth/login", cfg.IdentityLoginURL),
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func runtimeAuthMode(cfg appConfig) string {
	mode := strings.TrimSpace(cfg.AuthMode)
	if mode != "" {
		return mode
	}
	if cfg.RuntimeProfile == "managed" {
		if cfg.IdentityClientID != "" {
			return "oidc-pkce"
		}
		return "nebula-direct"
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
			w.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src %s; object-src 'none'; frame-ancestors 'none'", strings.Join(connectSources, " ")))
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
		cfg.IdentityLoginURL,
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
			Endpoint:    cfg.AuditSink,
			BearerToken: cfg.AuditToken,
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
	if cfg.IdentityHS256Secret != "" {
		return service.NewNebulaJWTIdentityVerifier(service.NebulaJWTIdentityConfig{
			Issuer: cfg.IdentityIssuer,
			Secret: cfg.IdentityHS256Secret,
		})
	}
	return service.NewJWKSIdentityVerifier(service.JWKSIdentityConfig{
		Issuer:  cfg.IdentityIssuer,
		JWKSURL: cfg.IdentityJWKSURL,
	})
}
