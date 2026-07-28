package service

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type failingArtworkStore struct{}

func (failingArtworkStore) Create(model.ArtworkDraft) (model.ArtworkDraft, error) {
	return model.ArtworkDraft{}, errors.New("database unavailable")
}
func (failingArtworkStore) Get(string) (model.ArtworkDraft, bool, error) {
	return model.ArtworkDraft{}, false, errors.New("database unavailable")
}
func (failingArtworkStore) ListByOwner(string) ([]model.ArtworkDraft, error) {
	return nil, errors.New("database unavailable")
}
func (failingArtworkStore) Update(model.ArtworkDraft) error {
	return errors.New("database unavailable")
}
func (failingArtworkStore) Delete(string) (bool, error) {
	return false, errors.New("database unavailable")
}

func TestArtworkServiceCreatesAndGetsDraft(t *testing.T) {
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer())

	draft, err := artworks.CreateDraft(model.CreateArtworkDraftRequest{
		OwnerUserID: "user-1",
		Layout: model.LayoutRequest{
			Text: "山水清音",
			Paper: model.PaperSpec{
				Format:   "doufang",
				WidthCM:  69,
				HeightCM: 68,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	got, ok, err := artworks.GetDraft(draft.ArtworkID)
	if err != nil {
		t.Fatalf("GetDraft(%q) error = %v", draft.ArtworkID, err)
	}
	if !ok {
		t.Fatalf("GetDraft(%q) not found", draft.ArtworkID)
	}
	if got.OwnerUserID != "user-1" {
		t.Fatalf("OwnerUserID = %q, want user-1", got.OwnerUserID)
	}
	if got.Layout.CharacterCount != 4 {
		t.Fatalf("CharacterCount = %d, want 4", got.Layout.CharacterCount)
	}
}

func TestArtworkServiceListsDraftsByOwner(t *testing.T) {
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer())

	_, err := artworks.CreateDraft(validDraftRequest("user-1", "山水"))
	if err != nil {
		t.Fatalf("CreateDraft(user-1) error = %v", err)
	}
	_, err = artworks.CreateDraft(validDraftRequest("user-2", "清音"))
	if err != nil {
		t.Fatalf("CreateDraft(user-2) error = %v", err)
	}

	items, err := artworks.ListDrafts("user-1")
	if err != nil {
		t.Fatalf("ListDrafts(user-1) error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(ListDrafts(user-1)) = %d, want 1", len(items))
	}
	if items[0].OwnerUserID != "user-1" {
		t.Fatalf("OwnerUserID = %q, want user-1", items[0].OwnerUserID)
	}
}

func TestArtworkServiceDeletesDraft(t *testing.T) {
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer())
	draft, err := artworks.CreateDraft(validDraftRequest("user-1", "山水"))
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	ok, err := artworks.DeleteDraft(draft.ArtworkID)
	if err != nil {
		t.Fatalf("DeleteDraft() error = %v", err)
	}
	if !ok {
		t.Fatal("DeleteDraft() = false, want true")
	}
	if _, ok, err := artworks.GetDraft(draft.ArtworkID); err != nil || ok {
		t.Fatal("GetDraft() found deleted draft")
	}
	if ok, err := artworks.DeleteDraft(draft.ArtworkID); err != nil || ok {
		t.Fatal("DeleteDraft() second call = true, want false")
	}
}

func TestArtworkServiceExportsSVG(t *testing.T) {
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer())
	draft, err := artworks.CreateDraft(validDraftRequest("user-1", "山水清音"))
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	export, err := artworks.ExportDraft(draft.ArtworkID, model.CreateExportRequest{
		Format:       "svg",
		TemplateType: "reference",
	})
	if err != nil {
		t.Fatalf("ExportDraft() error = %v", err)
	}

	if export.Format != "svg" {
		t.Fatalf("Format = %q, want svg", export.Format)
	}
	if export.ContentType != "image/svg+xml" {
		t.Fatalf("ContentType = %q, want image/svg+xml", export.ContentType)
	}
	if export.SHA256 == "" {
		t.Fatal("SHA256 is empty")
	}
	if !strings.Contains(export.InlineContent, "<svg") || !strings.Contains(export.InlineContent, "山") {
		t.Fatalf("InlineContent does not look like rendered SVG: %.80q", export.InlineContent)
	}
}

func TestArtworkServiceExportsPNG(t *testing.T) {
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer())
	draft, err := artworks.CreateDraft(validDraftRequest("user-1", "山水清音"))
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	export, err := artworks.ExportDraft(draft.ArtworkID, model.CreateExportRequest{
		Format:       "png",
		TemplateType: "reference",
	})
	if err != nil {
		t.Fatalf("ExportDraft(png) error = %v", err)
	}

	if export.Format != "png" {
		t.Fatalf("Format = %q, want png", export.Format)
	}
	if export.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", export.ContentType)
	}
	if export.InlineEncoding != "base64" {
		t.Fatalf("InlineEncoding = %q, want base64", export.InlineEncoding)
	}
	content, err := base64.StdEncoding.DecodeString(export.InlineContent)
	if err != nil {
		t.Fatalf("DecodeString(InlineContent) error = %v", err)
	}
	if len(content) < 8 || string(content[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("PNG signature = % x, want PNG", content[:min(8, len(content))])
	}
	if export.ByteSize != len(content) {
		t.Fatalf("ByteSize = %d, want %d", export.ByteSize, len(content))
	}
}

func TestFileArtworkStorePersistsDraftsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drafts.json")
	store, err := NewFileArtworkStore(path)
	if err != nil {
		t.Fatalf("NewFileArtworkStore() error = %v", err)
	}
	artworks := NewArtworkService(store, NewLayoutEngine(), NewSVGRenderer(), nil)

	draft, err := artworks.CreateDraft(validDraftRequest("user-1", "山水"))
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	reloadedStore, err := NewFileArtworkStore(path)
	if err != nil {
		t.Fatalf("NewFileArtworkStore(reload) error = %v", err)
	}
	reloaded, ok, err := reloadedStore.Get(draft.ArtworkID)
	if err != nil {
		t.Fatalf("Get(%q) after reload error = %v", draft.ArtworkID, err)
	}
	if !ok {
		t.Fatalf("Get(%q) after reload not found", draft.ArtworkID)
	}
	if reloaded.Layout.NormalizedText != "山水" {
		t.Fatalf("NormalizedText = %q, want 山水", reloaded.Layout.NormalizedText)
	}
}

func TestArtworkServiceWritesExportToArtifactStore(t *testing.T) {
	dir := t.TempDir()
	artifactStore := NewLocalArtifactStore(dir)
	artworks := NewArtworkService(NewInMemoryArtworkStore(), NewLayoutEngine(), NewSVGRenderer(), artifactStore)
	draft, err := artworks.CreateDraft(validDraftRequest("user-1", "山水"))
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	export, err := artworks.ExportDraft(draft.ArtworkID, model.CreateExportRequest{
		Format:       "svg",
		TemplateType: "reference",
	})
	if err != nil {
		t.Fatalf("ExportDraft() error = %v", err)
	}

	if export.StorageKey == "" {
		t.Fatal("StorageKey is empty")
	}
	if export.InlineContent != "" {
		t.Fatal("InlineContent should be empty when artifact store is configured")
	}
	content, err := os.ReadFile(filepath.Join(dir, export.StorageKey))
	if err != nil {
		t.Fatalf("ReadFile(export) error = %v", err)
	}
	if !strings.Contains(string(content), "<svg") {
		t.Fatalf("stored artifact does not contain svg: %.80q", string(content))
	}
}

func TestArtworkServicePropagatesPersistenceFailure(t *testing.T) {
	artworks := NewArtworkService(failingArtworkStore{}, NewLayoutEngine(), NewSVGRenderer())
	_, err := artworks.CreateDraft(validDraftRequest("user-1", "山水"))
	if !errors.Is(err, ErrPersistence) {
		t.Fatalf("CreateDraft() error = %v, want persistence error", err)
	}
	if _, err := artworks.ListDrafts("user-1"); !errors.Is(err, ErrPersistence) {
		t.Fatalf("ListDrafts() error = %v, want persistence error", err)
	}
}

func validDraftRequest(owner, text string) model.CreateArtworkDraftRequest {
	return model.CreateArtworkDraftRequest{
		OwnerUserID: owner,
		Layout: model.LayoutRequest{
			Text: text,
			Paper: model.PaperSpec{
				Format:   "doufang",
				WidthCM:  69,
				HeightCM: 68,
			},
		},
	}
}
