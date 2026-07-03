package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

type Handler struct {
	catalog       service.GlyphCatalog
	layout        *service.LayoutEngine
	artworks      *service.ArtworkService
	learning      *service.LearningService
	auth          *service.AuthService
	identity      service.IdentityVerifier
	audit         service.AuditLogger
	glyphRenderer *service.GlyphImageRenderer
}

func New(catalog service.GlyphCatalog, layout *service.LayoutEngine, artworks *service.ArtworkService, learning *service.LearningService, auth *service.AuthService, audit service.AuditLogger, identity ...service.IdentityVerifier) *Handler {
	if audit == nil {
		audit = service.NoopAuditLogger{}
	}
	activeIdentity := service.IdentityVerifier(auth)
	if len(identity) > 0 && identity[0] != nil {
		activeIdentity = identity[0]
	}
	return &Handler{
		catalog:       catalog,
		layout:        layout,
		artworks:      artworks,
		learning:      learning,
		auth:          auth,
		identity:      activeIdentity,
		audit:         audit,
		glyphRenderer: service.NewGlyphImageRenderer("", ""),
	}
}

func (h *Handler) SetGlyphRenderer(renderer *service.GlyphImageRenderer) {
	if renderer != nil {
		h.glyphRenderer = renderer
	}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/health", h.Health)
	r.Get("/api/v1/calligraphy/glyphs/search", h.SearchGlyphs)
	r.Post("/api/v1/calligraphy/auth/register", h.Register)
	r.Post("/api/v1/calligraphy/auth/login", h.Login)
	r.Post("/api/v1/calligraphy/auth/logout", h.Logout)
	r.Get("/api/v1/calligraphy/auth/me", h.CurrentUser)
	r.Get("/api/v1/calligraphy/glyphs/presets", h.ListGlyphPresets)
	r.Get("/api/v1/calligraphy/glyphs/{glyphID}", h.GetGlyphDetail)
	r.Get("/api/v1/calligraphy/glyphs/{glyphID}/render.png", h.RenderGlyphPNG)
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
	for index := range items {
		items[index] = h.withGlyphRenderAsset(r, items[index])
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	session, err := h.auth.Register(req)
	if err != nil {
		h.recordAudit("auth.register", "", "", "failure")
		writeError(w, http.StatusBadRequest, "invalid_auth_request", err.Error())
		return
	}
	h.recordAudit("auth.register", session.User.UserID, session.User.UserID, "success")
	writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	session, err := h.auth.Login(req)
	if err != nil {
		h.recordAudit("auth.login", "", "", "failure")
		writeError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
		return
	}
	h.recordAudit("auth.login", session.User.UserID, session.User.UserID, "success")
	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := h.identity.CurrentUser(bearerToken(r.Header.Get("Authorization")))
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.auth.Logout(bearerToken(r.Header.Get("Authorization"))) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
		return
	}
	h.recordAudit("auth.logout", "", "", "success")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetGlyphDetail(w http.ResponseWriter, r *http.Request) {
	detail, ok := h.catalog.GetDetail(chi.URLParam(r, "glyphID"))
	if !ok {
		writeError(w, http.StatusNotFound, "glyph_not_found", "glyph not found")
		return
	}
	detail.Glyph = h.withGlyphRenderAsset(r, detail.Glyph)
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) ListGlyphPresets(w http.ResponseWriter, r *http.Request) {
	groups := h.catalog.ListPresetGroups(r.URL.Query().Get("style"))
	for groupIndex := range groups {
		for glyphIndex := range groups[groupIndex].Glyphs {
			groups[groupIndex].Glyphs[glyphIndex] = h.withGlyphRenderAsset(r, groups[groupIndex].Glyphs[glyphIndex])
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": groups})
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
	h.enrichLayoutSlots(r, &result)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RenderGlyphPNG(w http.ResponseWriter, r *http.Request) {
	detail, ok := h.catalog.GetDetail(chi.URLParam(r, "glyphID"))
	if !ok {
		writeError(w, http.StatusNotFound, "glyph_not_found", "glyph not found")
		return
	}
	content, err := h.glyphRenderer.RenderPNG(detail.Glyph, service.GlyphRenderOptions{
		Grid: r.URL.Query().Get("grid"),
		Size: 512,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "glyph_render_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) CreateArtworkDraft(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var req model.CreateArtworkDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	if req.OwnerUserID != "" && req.OwnerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "owner_user_id must match authenticated user")
		return
	}
	req.OwnerUserID = user.UserID

	draft, err := h.artworks.CreateDraft(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_artwork_request", err.Error())
		return
	}
	h.recordAudit("artwork.create", user.UserID, draft.ArtworkID, "success")
	writeJSON(w, http.StatusCreated, draft)
}

func (h *Handler) ListArtworkDrafts(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	ownerUserID := r.URL.Query().Get("owner_user_id")
	if ownerUserID != "" && ownerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "owner_user_id must match authenticated user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": h.artworks.ListDrafts(user.UserID)})
}

func (h *Handler) GetArtworkDraft(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	artworkID := chi.URLParam(r, "artworkID")
	draft, ok := h.artworks.GetDraft(artworkID)
	if !ok {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	if draft.OwnerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "artwork owner must match authenticated user")
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (h *Handler) DeleteArtworkDraft(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	artworkID := chi.URLParam(r, "artworkID")
	draft, ok := h.artworks.GetDraft(artworkID)
	if !ok {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	if draft.OwnerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "artwork owner must match authenticated user")
		return
	}
	if !h.artworks.DeleteDraft(artworkID) {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	h.recordAudit("artwork.delete", user.UserID, artworkID, "success")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ExportArtworkDraft(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	artworkID := chi.URLParam(r, "artworkID")
	draft, ok := h.artworks.GetDraft(artworkID)
	if !ok {
		writeError(w, http.StatusNotFound, "artwork_not_found", "artwork draft not found")
		return
	}
	if draft.OwnerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "artwork owner must match authenticated user")
		return
	}

	var req model.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	export, err := h.artworks.ExportDraft(artworkID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_export_request", err.Error())
		return
	}
	h.recordAudit("artwork.export", user.UserID, export.ExportID, "success")
	writeJSON(w, http.StatusCreated, export)
}

func (h *Handler) GetLearningProfile(w http.ResponseWriter, r *http.Request) {
	ownerUserID := chi.URLParam(r, "ownerUserID")
	if _, ok := h.requireOwner(w, r, ownerUserID); !ok {
		return
	}
	writeJSON(w, http.StatusOK, h.learning.GetProfile(ownerUserID))
}

func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ownerUserID := chi.URLParam(r, "ownerUserID")
	if _, ok := h.requireOwner(w, r, ownerUserID); !ok {
		return
	}

	var req model.CreateFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	favorite, err := h.learning.AddFavorite(ownerUserID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_favorite_request", err.Error())
		return
	}
	h.recordAudit("favorite.create", ownerUserID, favorite.GlyphID, "success")
	writeJSON(w, http.StatusCreated, favorite)
}

func (h *Handler) DeleteFavorite(w http.ResponseWriter, r *http.Request) {
	ownerUserID := chi.URLParam(r, "ownerUserID")
	if _, ok := h.requireOwner(w, r, ownerUserID); !ok {
		return
	}
	if !h.learning.DeleteFavorite(ownerUserID, chi.URLParam(r, "glyphID")) {
		writeError(w, http.StatusNotFound, "favorite_not_found", "favorite glyph not found")
		return
	}
	h.recordAudit("favorite.delete", ownerUserID, chi.URLParam(r, "glyphID"), "success")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RecordPractice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ownerUserID := chi.URLParam(r, "ownerUserID")
	if _, ok := h.requireOwner(w, r, ownerUserID); !ok {
		return
	}

	var req model.CreatePracticeRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	record, err := h.learning.RecordPractice(ownerUserID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_practice_request", err.Error())
		return
	}
	h.recordAudit("practice.record", ownerUserID, record.PracticeID, "success")
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

func (h *Handler) enrichLayoutSlots(r *http.Request, layout *model.LayoutResult) {
	for index := range layout.Slots {
		matches := h.catalog.Search(service.GlyphSearchParams{
			Character:  layout.Slots[index].Character,
			Style:      layout.Style,
			CopybookID: layout.CopybookID,
		})
		if len(matches) == 0 {
			matches = h.catalog.Search(service.GlyphSearchParams{
				Character: layout.Slots[index].Character,
				Style:     layout.Style,
			})
		}
		if len(matches) == 0 {
			continue
		}
		layout.Slots[index].RenderAsset = h.withGlyphRenderAsset(r, matches[0]).RenderAsset
	}
}

func (h *Handler) withGlyphRenderAsset(r *http.Request, glyph model.Glyph) model.Glyph {
	if glyph.RenderAsset.URL != "" {
		return glyph
	}
	glyph.RenderAsset = model.RenderAsset{
		URL:         absoluteURL(r, "/api/v1/calligraphy/glyphs/"+url.PathEscape(glyph.GlyphID)+"/render.png"),
		ContentType: "image/png",
		Width:       512,
		Height:      512,
		Source:      "server_font",
	}
	return glyph
}

func absoluteURL(r *http.Request, path string) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host + path
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	user, ok := h.identity.CurrentUser(bearerToken(r.Header.Get("Authorization")))
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
		return model.User{}, false
	}
	return user, true
}

func (h *Handler) requireOwner(w http.ResponseWriter, r *http.Request, ownerUserID string) (model.User, bool) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return model.User{}, false
	}
	if ownerUserID != user.UserID {
		writeError(w, http.StatusForbidden, "forbidden_owner", "owner_user_id must match authenticated user")
		return model.User{}, false
	}
	return user, true
}

func bearerToken(header string) string {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func (h *Handler) recordAudit(action, actorID, resourceID, outcome string) {
	_ = h.audit.Record(service.AuditEvent{
		Action:     action,
		ActorID:    actorID,
		ResourceID: resourceID,
		Outcome:    outcome,
	})
}
