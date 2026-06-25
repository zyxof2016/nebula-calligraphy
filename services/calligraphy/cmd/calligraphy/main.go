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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	router := chi.NewRouter()
	router.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer)
	handler.RegisterRoutes(router, handler.New(service.NewInMemoryGlyphCatalog(), service.NewLayoutEngine()))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("calligraphy service listening on :%s", port)
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
