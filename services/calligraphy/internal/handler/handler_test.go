package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

func TestHealth(t *testing.T) {
	router := newTestRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSearchGlyphs(t *testing.T) {
	router := newTestRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/glyphs/search?character=山&style=ou", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []model.Glyph `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(payload.Items))
	}
}

func TestGetGlyphDetail(t *testing.T) {
	router := newTestRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/glyphs/ou-jiuchenggong-shan", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var payload model.GlyphDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Glyph.GlyphID != "ou-jiuchenggong-shan" {
		t.Fatalf("GlyphID = %q, want ou-jiuchenggong-shan", payload.Glyph.GlyphID)
	}
	if len(payload.PracticeTemplates) == 0 {
		t.Fatal("PracticeTemplates is empty")
	}
}

func TestListGlyphPresetGroups(t *testing.T) {
	router := newTestRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/glyphs/presets?style=yan", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []model.GlyphPresetGroup `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) < 5 {
		t.Fatalf("len(items) = %d, want at least 5", len(payload.Items))
	}
	if payload.Items[0].Glyphs[0].Style != "yan" {
		t.Fatalf("style = %q, want yan", payload.Items[0].Glyphs[0].Style)
	}
}

func TestPreviewLayout(t *testing.T) {
	router := newTestRouter()
	body := []byte(`{"text":"山水清音","style":"ou","paper":{"format":"doufang","width_cm":69,"height_cm":68},"direction":"vertical_rtl","margin_cm":3}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/layouts/preview", bytes.NewReader(body))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var payload model.LayoutResult
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.CharacterCount != 4 {
		t.Fatalf("CharacterCount = %d, want 4", payload.CharacterCount)
	}
}

func TestCreateAndGetArtworkDraft(t *testing.T) {
	router := newTestRouter()
	body := []byte(`{"owner_user_id":"user-1","layout":{"text":"山水清音","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", createRec.Code, createRec.Body.String())
	}

	var created model.ArtworkDraft
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}
	if created.ArtworkID == "" {
		t.Fatal("ArtworkID is empty")
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200: %s", getRec.Code, getRec.Body.String())
	}
}

func TestListArtworkDraftsByOwner(t *testing.T) {
	router := newTestRouter()
	body := []byte(`{"owner_user_id":"user-1","layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	router.ServeHTTP(httptest.NewRecorder(), createReq)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts?owner_user_id=user-1", nil)
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", listRec.Code, listRec.Body.String())
	}

	var payload struct {
		Items []model.ArtworkDraft `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(payload.Items))
	}
}

func TestDeleteArtworkDraft(t *testing.T) {
	router := newTestRouter()
	body := []byte(`{"owner_user_id":"user-1","layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	router.ServeHTTP(createRec, createReq)

	var created model.ArtworkDraft
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", deleteRec.Code, deleteRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d, want 404", getRec.Code)
	}
}

func TestExportArtworkDraftSVG(t *testing.T) {
	router := newTestRouter()
	body := []byte(`{"owner_user_id":"user-1","layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	router.ServeHTTP(createRec, createReq)

	var created model.ArtworkDraft
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	exportBody := []byte(`{"format":"svg","template_type":"reference"}`)
	exportRec := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID+"/exports", bytes.NewReader(exportBody))
	router.ServeHTTP(exportRec, exportReq)

	if exportRec.Code != http.StatusCreated {
		t.Fatalf("export status = %d, want 201: %s", exportRec.Code, exportRec.Body.String())
	}

	var export model.ExportRecord
	if err := json.Unmarshal(exportRec.Body.Bytes(), &export); err != nil {
		t.Fatalf("json.Unmarshal(export) error = %v", err)
	}
	if export.ContentType != "image/svg+xml" {
		t.Fatalf("ContentType = %q, want image/svg+xml", export.ContentType)
	}
	if !strings.Contains(export.InlineContent, "<svg") {
		t.Fatalf("InlineContent does not contain svg: %.80q", export.InlineContent)
	}
}

func TestLearningProfileRoutes(t *testing.T) {
	router := newTestRouter()

	favoriteRec := httptest.NewRecorder()
	favoriteReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/users/user-1/favorites", bytes.NewReader([]byte(`{"glyph_id":"ou-common-人"}`)))
	router.ServeHTTP(favoriteRec, favoriteReq)

	if favoriteRec.Code != http.StatusCreated {
		t.Fatalf("favorite status = %d, want 201: %s", favoriteRec.Code, favoriteRec.Body.String())
	}

	practiceRec := httptest.NewRecorder()
	practiceReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/users/user-1/practice", bytes.NewReader([]byte(`{"glyph_id":"ou-common-人","template_type":"copy","grid_type":"mi"}`)))
	router.ServeHTTP(practiceRec, practiceReq)

	if practiceRec.Code != http.StatusCreated {
		t.Fatalf("practice status = %d, want 201: %s", practiceRec.Code, practiceRec.Body.String())
	}

	profileRec := httptest.NewRecorder()
	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/users/user-1/learning", nil)
	router.ServeHTTP(profileRec, profileReq)

	if profileRec.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200: %s", profileRec.Code, profileRec.Body.String())
	}

	var profile model.LearningProfile
	if err := json.Unmarshal(profileRec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("json.Unmarshal(profile) error = %v", err)
	}
	if profile.PracticeCount != 1 {
		t.Fatalf("PracticeCount = %d, want 1", profile.PracticeCount)
	}
	if len(profile.Favorites) != 1 {
		t.Fatalf("len(Favorites) = %d, want 1", len(profile.Favorites))
	}
}

func newTestRouter() http.Handler {
	router := chi.NewRouter()
	layout := service.NewLayoutEngine()
	catalog := service.NewInMemoryGlyphCatalog()
	RegisterRoutes(router, New(
		catalog,
		layout,
		service.NewArtworkService(service.NewInMemoryArtworkStore(), layout, service.NewSVGRenderer()),
		service.NewLearningService(service.NewInMemoryLearningStore(), catalog),
	))
	return router
}
