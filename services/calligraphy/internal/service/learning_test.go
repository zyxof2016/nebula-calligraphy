package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

func TestLearningServiceFavoritesAndPracticeProfile(t *testing.T) {
	catalog := NewInMemoryGlyphCatalog()
	svc := NewLearningService(NewInMemoryLearningStore(), catalog)
	svc.now = func() time.Time { return time.Date(2026, 6, 25, 8, 30, 0, 0, time.UTC) }

	favorite, err := svc.AddFavorite("learner-1", model.CreateFavoriteRequest{GlyphID: "ou-common-人"})
	if err != nil {
		t.Fatalf("AddFavorite() error = %v", err)
	}
	if favorite.Character != "人" || favorite.Style != "ou" {
		t.Fatalf("favorite = %#v, want enriched ou 人 glyph", favorite)
	}

	practice, err := svc.RecordPractice("learner-1", model.CreatePracticeRecordRequest{
		GlyphID:      "ou-common-人",
		TemplateType: "copy",
		GridType:     "mi",
	})
	if err != nil {
		t.Fatalf("RecordPractice() error = %v", err)
	}
	if practice.PracticeID == "" {
		t.Fatal("PracticeID is empty")
	}

	profile := svc.GetProfile("learner-1")
	if profile.PracticeCount != 1 {
		t.Fatalf("PracticeCount = %d, want 1", profile.PracticeCount)
	}
	if len(profile.Favorites) != 1 || profile.Favorites[0].GlyphID != "ou-common-人" {
		t.Fatalf("Favorites = %#v, want ou-common-人", profile.Favorites)
	}
	if len(profile.RecentPractice) != 1 || profile.RecentPractice[0].GlyphID != "ou-common-人" {
		t.Fatalf("RecentPractice = %#v, want ou-common-人", profile.RecentPractice)
	}
}

func TestFileLearningStorePersistsProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	catalog := NewInMemoryGlyphCatalog()

	store, err := NewFileLearningStore(path)
	if err != nil {
		t.Fatalf("NewFileLearningStore() error = %v", err)
	}
	svc := NewLearningService(store, catalog)
	if _, err := svc.AddFavorite("learner-1", model.CreateFavoriteRequest{GlyphID: "yan-common-山"}); err != nil {
		t.Fatalf("AddFavorite() error = %v", err)
	}
	if _, err := svc.RecordPractice("learner-1", model.CreatePracticeRecordRequest{GlyphID: "yan-common-山"}); err != nil {
		t.Fatalf("RecordPractice() error = %v", err)
	}

	reloadedStore, err := NewFileLearningStore(path)
	if err != nil {
		t.Fatalf("NewFileLearningStore(reload) error = %v", err)
	}
	reloaded := NewLearningService(reloadedStore, catalog).GetProfile("learner-1")
	if len(reloaded.Favorites) != 1 {
		t.Fatalf("len(Favorites) = %d, want 1", len(reloaded.Favorites))
	}
	if reloaded.PracticeCount != 1 {
		t.Fatalf("PracticeCount = %d, want 1", reloaded.PracticeCount)
	}
}
