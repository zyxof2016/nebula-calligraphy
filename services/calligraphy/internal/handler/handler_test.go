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
	session := registerTestSession(t, router, "learner-a")
	body := []byte(`{"layout":{"text":"山水清音","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	addBearer(createReq, session.Token)
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
	if created.OwnerUserID != session.User.UserID {
		t.Fatalf("OwnerUserID = %q, want authenticated user %q", created.OwnerUserID, session.User.UserID)
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	addBearer(getReq, session.Token)
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200: %s", getRec.Code, getRec.Body.String())
	}
}

func TestListArtworkDraftsByOwner(t *testing.T) {
	router := newTestRouter()
	session := registerTestSession(t, router, "learner-b")
	body := []byte(`{"layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	addBearer(createReq, session.Token)
	router.ServeHTTP(httptest.NewRecorder(), createReq)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts?owner_user_id="+session.User.UserID, nil)
	addBearer(listReq, session.Token)
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
	session := registerTestSession(t, router, "learner-c")
	body := []byte(`{"layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	addBearer(createReq, session.Token)
	router.ServeHTTP(createRec, createReq)

	var created model.ArtworkDraft
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	addBearer(deleteReq, session.Token)
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", deleteRec.Code, deleteRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID, nil)
	addBearer(getReq, session.Token)
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d, want 404", getRec.Code)
	}
}

func TestExportArtworkDraftSVG(t *testing.T) {
	router := newTestRouter()
	session := registerTestSession(t, router, "learner-d")
	body := []byte(`{"layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader(body))
	addBearer(createReq, session.Token)
	router.ServeHTTP(createRec, createReq)

	var created model.ArtworkDraft
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	exportBody := []byte(`{"format":"svg","template_type":"reference"}`)
	exportRec := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts/"+created.ArtworkID+"/exports", bytes.NewReader(exportBody))
	addBearer(exportReq, session.Token)
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
	session := registerTestSession(t, router, "learner-e")

	favoriteRec := httptest.NewRecorder()
	favoriteReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/users/"+session.User.UserID+"/favorites", bytes.NewReader([]byte(`{"glyph_id":"ou-common-人"}`)))
	addBearer(favoriteReq, session.Token)
	router.ServeHTTP(favoriteRec, favoriteReq)

	if favoriteRec.Code != http.StatusCreated {
		t.Fatalf("favorite status = %d, want 201: %s", favoriteRec.Code, favoriteRec.Body.String())
	}

	practiceRec := httptest.NewRecorder()
	practiceReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/users/"+session.User.UserID+"/practice", bytes.NewReader([]byte(`{"glyph_id":"ou-common-人","template_type":"copy","grid_type":"mi"}`)))
	addBearer(practiceReq, session.Token)
	router.ServeHTTP(practiceRec, practiceReq)

	if practiceRec.Code != http.StatusCreated {
		t.Fatalf("practice status = %d, want 201: %s", practiceRec.Code, practiceRec.Body.String())
	}

	profileRec := httptest.NewRecorder()
	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/users/"+session.User.UserID+"/learning", nil)
	addBearer(profileReq, session.Token)
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

func TestAuthRegisterLoginAndMeRoutes(t *testing.T) {
	router := newTestRouter()

	registerRec := httptest.NewRecorder()
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/auth/register", bytes.NewReader([]byte(`{"username":"learner","password":"secret123"}`)))
	router.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201: %s", registerRec.Code, registerRec.Body.String())
	}

	var registered model.AuthSession
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("json.Unmarshal(register) error = %v", err)
	}
	if registered.Token == "" || registered.User.UserID == "" || registered.User.Username != "learner" {
		t.Fatalf("registered = %#v, want token and learner user", registered)
	}

	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+registered.Token)
	router.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want 200: %s", meRec.Code, meRec.Body.String())
	}

	var me model.User
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil {
		t.Fatalf("json.Unmarshal(me) error = %v", err)
	}
	if me.UserID != registered.User.UserID {
		t.Fatalf("me UserID = %q, want %q", me.UserID, registered.User.UserID)
	}

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/auth/login", bytes.NewReader([]byte(`{"username":"learner","password":"secret123"}`)))
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", loginRec.Code, loginRec.Body.String())
	}

	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+registered.Token)
	router.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204: %s", logoutRec.Code, logoutRec.Body.String())
	}

	afterLogoutRec := httptest.NewRecorder()
	afterLogoutReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/auth/me", nil)
	afterLogoutReq.Header.Set("Authorization", "Bearer "+registered.Token)
	router.ServeHTTP(afterLogoutRec, afterLogoutReq)

	if afterLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401", afterLogoutRec.Code)
	}
}

func TestUserAssetsRequireAuthenticatedOwner(t *testing.T) {
	router := newTestRouter()
	session := registerTestSession(t, router, "owner-a")

	unauthRec := httptest.NewRecorder()
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/calligraphy/artworks/drafts?owner_user_id="+session.User.UserID, nil)
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth list status = %d, want 401", unauthRec.Code)
	}

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/artworks/drafts", bytes.NewReader([]byte(`{"owner_user_id":"user-other","layout":{"text":"山水","paper":{"format":"doufang","width_cm":69,"height_cm":68}}}`)))
	addBearer(createReq, session.Token)
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusForbidden {
		t.Fatalf("spoof create status = %d, want 403: %s", createRec.Code, createRec.Body.String())
	}

	favoriteRec := httptest.NewRecorder()
	favoriteReq := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/users/user-other/favorites", bytes.NewReader([]byte(`{"glyph_id":"ou-common-人"}`)))
	addBearer(favoriteReq, session.Token)
	router.ServeHTTP(favoriteRec, favoriteReq)
	if favoriteRec.Code != http.StatusForbidden {
		t.Fatalf("spoof favorite status = %d, want 403: %s", favoriteRec.Code, favoriteRec.Body.String())
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
		service.NewAuthService(service.NewInMemoryAuthStore()),
		service.NoopAuditLogger{},
	))
	return router
}

func registerTestSession(t *testing.T, router http.Handler, username string) model.AuthSession {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calligraphy/auth/register", bytes.NewReader([]byte(`{"username":"`+username+`","password":"secret123"}`)))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var session model.AuthSession
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("json.Unmarshal(session) error = %v", err)
	}
	return session
}

func addBearer(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}
