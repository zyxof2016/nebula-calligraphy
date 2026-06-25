package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

type Handler struct {
	catalog service.GlyphCatalog
	layout  *service.LayoutEngine
}

func New(catalog service.GlyphCatalog, layout *service.LayoutEngine) *Handler {
	return &Handler{catalog: catalog, layout: layout}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/health", h.Health)
	r.Get("/api/v1/calligraphy/glyphs/search", h.SearchGlyphs)
	r.Post("/api/v1/calligraphy/layouts/preview", h.PreviewLayout)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "calligraphy",
	})
}

func (h *Handler) SearchGlyphs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items := h.catalog.Search(service.GlyphSearchParams{
		Character:  query.Get("character"),
		Style:      query.Get("style"),
		CopybookID: query.Get("copybook_id"),
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) PreviewLayout(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.LayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	result, err := h.layout.Preview(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_layout_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}
