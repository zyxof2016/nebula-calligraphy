package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

type Handler struct {
	catalog  service.GlyphCatalog
	layout   *service.LayoutEngine
	artworks *service.ArtworkService
	learning *service.LearningService
}

func New(catalog service.GlyphCatalog, layout *service.LayoutEngine, artworks *service.ArtworkService, learning *service.LearningService) *Handler {
	return &Handler{catalog: catalog, layout: layout, artworks: artworks, learning: learning}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/health", h.Health)
	r.Get("/api/v1/calligraphy/glyphs/search", h.SearchGlyphs)
	r.Get("/api/v1/calligraphy/glyphs/presets", h.ListGlyphPresets)
	r.Get("/api/v1/calligraphy/glyphs/{glyphID}", h.GetGlyphDetail)
	r.Post("/api/v1/calligraphy/layouts/preview", h.PreviewLayout)
	r.Get("/api/v1/calligraphy/artworks/drafts", h.ListArtworkDrafts)
	r.Post("/api/v1/calligraphy/artworks/drafts", h.CreateArtworkDraft)
	r.Get("/api/v1/calligraphy/artworks/drafts/{artworkID}", h.GetArtworkDraft)
	r.Delete("/api/v1/calligraphy/artworks/drafts/{artworkID}", h.DeleteArtworkDraft)
	r.Post("/api/v1/calligraphy/artworks/drafts/{artworkID}/exports", h.ExportArtworkDraft)
	r.Get("/api/v1/calligraphy/users/{ownerUserID}/learning", h.GetLearningProfile)
	r.Post("/api/v1/calligraphy/users/{ownerUserID}/favorites", h.AddFavorite)
	r.Delete("/api/v1/calligraphy/users/{ownerUserID}/favorites/{glyphID}", h.DeleteFavorite)
	r.Post("/api/v1/calligraphy/users/{ownerUserID}/practice", h.RecordPractice)
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

func (h *Handler) GetGlyphDetail(w http.ResponseWriter, r *http.Request) {
	detail, ok := h.catalog.GetDetail(chi.URLParam(r, "glyphID"))
	if !ok {
		writeError(w, http.StatusNotFound, "glyph_not_found", "glyph not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) ListGlyphPresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.catalog.ListPresetGroups(r.URL.Query().Get("style"))})
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

func (h *Handler) CreateArtworkDraft(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.CreateArtworkDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	draft, err := h.artworks.CreateDraft(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_artwork_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, draft)
}

func (h *Handler) ListArtworkDrafts(w http.ResponseWriter, r *http.Request) {
	ownerUserID := r.URL.Query().Get("owner_user_id")
	writeJSON(w, http.StatusOK, map[string]any{"items": h.artworks.ListDrafts(ownerUserID)})
}

func (h *Handler) GetArtworkDraft(w http.ResponseWriter, r *http.Request) {
	artworkID := chi.URLParam(r, "artworkID")
	draft, ok := h.artworks.GetDraft(artworkID)
	if !ok {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (h *Handler) DeleteArtworkDraft(w http.ResponseWriter, r *http.Request) {
	artworkID := chi.URLParam(r, "artworkID")
	if !h.artworks.DeleteDraft(artworkID) {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ExportArtworkDraft(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	export, err := h.artworks.ExportDraft(chi.URLParam(r, "artworkID"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_export_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, export)
}

func (h *Handler) GetLearningProfile(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.learning.GetProfile(chi.URLParam(r, "ownerUserID")))
}

func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.CreateFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	favorite, err := h.learning.AddFavorite(chi.URLParam(r, "ownerUserID"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_favorite_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, favorite)
}

func (h *Handler) DeleteFavorite(w http.ResponseWriter, r *http.Request) {
	if !h.learning.DeleteFavorite(chi.URLParam(r, "ownerUserID"), chi.URLParam(r, "glyphID")) {
		writeError(w, http.StatusNotFound, "favorite_not_found", "favorite glyph not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RecordPractice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.CreatePracticeRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	record, err := h.learning.RecordPractice(chi.URLParam(r, "ownerUserID"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_practice_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, record)
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
