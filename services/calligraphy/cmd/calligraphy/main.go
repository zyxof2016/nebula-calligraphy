package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	Port         string
	DataFile     string
	LearningFile string
	ExportDir    string
	WebDir       string
}

func loadConfig() appConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	return appConfig{
		Port:         port,
		DataFile:     os.Getenv("CALLIGRAPHY_DATA_FILE"),
		LearningFile: os.Getenv("CALLIGRAPHY_LEARNING_FILE"),
		ExportDir:    os.Getenv("CALLIGRAPHY_EXPORT_DIR"),
		WebDir:       os.Getenv("CALLIGRAPHY_WEB_DIR"),
	}
}

func newRouter(cfg appConfig) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer)

	layout := service.NewLayoutEngine()
	catalog := service.NewInMemoryGlyphCatalog()
	artworkStore, err := newArtworkStore(cfg.DataFile)
	if err != nil {
		return nil, err
	}
	learningStore, err := newLearningStore(cfg.LearningFile)
	if err != nil {
		return nil, err
	}
	handler.RegisterRoutes(router, handler.New(
		catalog,
		layout,
		service.NewArtworkService(artworkStore, layout, service.NewSVGRenderer(), newArtifactStore(cfg.ExportDir)),
		service.NewLearningService(learningStore, catalog),
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

func newArtworkStore(dataFile string) (service.ArtworkStore, error) {
	if dataFile == "" {
		return service.NewInMemoryArtworkStore(), nil
	}
	return service.NewFileArtworkStore(dataFile)
}

func newLearningStore(learningFile string) (service.LearningStore, error) {
	if learningFile == "" {
		return service.NewInMemoryLearningStore(), nil
	}
	return service.NewFileLearningStore(learningFile)
}

func newArtifactStore(exportDir string) service.ArtifactStore {
	if exportDir == "" {
		return nil
	}
	return service.NewLocalArtifactStore(exportDir)
}
