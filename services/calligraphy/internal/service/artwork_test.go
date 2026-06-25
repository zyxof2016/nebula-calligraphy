package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

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

	got, ok := artworks.GetDraft(draft.ArtworkID)
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

	items := artworks.ListDrafts("user-1")
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

	if ok := artworks.DeleteDraft(draft.ArtworkID); !ok {
		t.Fatal("DeleteDraft() = false, want true")
	}
	if _, ok := artworks.GetDraft(draft.ArtworkID); ok {
		t.Fatal("GetDraft() found deleted draft")
	}
	if ok := artworks.DeleteDraft(draft.ArtworkID); ok {
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
	reloaded, ok := reloadedStore.Get(draft.ArtworkID)
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
