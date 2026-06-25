package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type ArtworkStore interface {
	Create(draft model.ArtworkDraft) model.ArtworkDraft
	Get(artworkID string) (model.ArtworkDraft, bool)
	ListByOwner(ownerUserID string) []model.ArtworkDraft
	Update(draft model.ArtworkDraft) model.ArtworkDraft
	Delete(artworkID string) bool
}

type InMemoryArtworkStore struct {
	mu     sync.RWMutex
	next   int
	drafts map[string]model.ArtworkDraft
}

type fileArtworkState struct {
	Next   int                           `json:"next"`
	Drafts map[string]model.ArtworkDraft `json:"drafts"`
}

type FileArtworkStore struct {
	mu     sync.RWMutex
	path   string
	next   int
	drafts map[string]model.ArtworkDraft
}

func NewFileArtworkStore(path string) (*FileArtworkStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("artwork store path is required")
	}
	store := &FileArtworkStore{path: path, drafts: make(map[string]model.ArtworkDraft)}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileArtworkStore) Create(draft model.ArtworkDraft) model.ArtworkDraft {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	draft.ArtworkID = fmt.Sprintf("artwork-%06d", s.next)
	s.drafts[draft.ArtworkID] = draft
	_ = s.persistLocked()
	return draft
}

func (s *FileArtworkStore) Get(artworkID string) (model.ArtworkDraft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	draft, ok := s.drafts[artworkID]
	return draft, ok
}

func (s *FileArtworkStore) ListByOwner(ownerUserID string) []model.ArtworkDraft {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.ArtworkDraft, 0)
	for _, draft := range s.drafts {
		if draft.OwnerUserID == ownerUserID {
			items = append(items, draft)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items
}

func (s *FileArtworkStore) Update(draft model.ArtworkDraft) model.ArtworkDraft {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.drafts[draft.ArtworkID] = draft
	_ = s.persistLocked()
	return draft
}

func (s *FileArtworkStore) Delete(artworkID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.drafts[artworkID]; !ok {
		return false
	}
	delete(s.drafts, artworkID)
	_ = s.persistLocked()
	return true
}

func (s *FileArtworkStore) load() error {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var state fileArtworkState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}
	s.next = state.Next
	if state.Drafts != nil {
		s.drafts = state.Drafts
	}
	return nil
}

func (s *FileArtworkStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(fileArtworkState{Next: s.next, Drafts: s.drafts}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

type ArtifactStore interface {
	Save(export model.ExportRecord, content string) (string, error)
}

type LocalArtifactStore struct {
	dir string
}

func NewLocalArtifactStore(dir string) *LocalArtifactStore {
	return &LocalArtifactStore{dir: dir}
}

func (s *LocalArtifactStore) Save(export model.ExportRecord, content string) (string, error) {
	if strings.TrimSpace(s.dir) == "" {
		return "", errors.New("artifact directory is required")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	key := filepath.Join(export.ArtworkID, export.ExportID+"."+export.Format)
	path := filepath.Join(s.dir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return key, os.WriteFile(path, []byte(content), 0o644)
}

func NewInMemoryArtworkStore() *InMemoryArtworkStore {
	return &InMemoryArtworkStore{drafts: make(map[string]model.ArtworkDraft)}
}

func (s *InMemoryArtworkStore) Create(draft model.ArtworkDraft) model.ArtworkDraft {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	draft.ArtworkID = fmt.Sprintf("artwork-%06d", s.next)
	s.drafts[draft.ArtworkID] = draft
	return draft
}

func (s *InMemoryArtworkStore) Get(artworkID string) (model.ArtworkDraft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	draft, ok := s.drafts[artworkID]
	return draft, ok
}

func (s *InMemoryArtworkStore) ListByOwner(ownerUserID string) []model.ArtworkDraft {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.ArtworkDraft, 0)
	for _, draft := range s.drafts {
		if draft.OwnerUserID == ownerUserID {
			items = append(items, draft)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items
}

func (s *InMemoryArtworkStore) Update(draft model.ArtworkDraft) model.ArtworkDraft {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.drafts[draft.ArtworkID] = draft
	return draft
}

func (s *InMemoryArtworkStore) Delete(artworkID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.drafts[artworkID]; !ok {
		return false
	}
	delete(s.drafts, artworkID)
	return true
}

type ArtworkService struct {
	store         ArtworkStore
	layout        *LayoutEngine
	renderer      *SVGRenderer
	artifactStore ArtifactStore
	now           func() time.Time
}

func NewArtworkService(store ArtworkStore, layout *LayoutEngine, renderer *SVGRenderer, artifactStore ...ArtifactStore) *ArtworkService {
	var artifacts ArtifactStore
	if len(artifactStore) > 0 {
		artifacts = artifactStore[0]
	}
	return &ArtworkService{
		store:         store,
		layout:        layout,
		renderer:      renderer,
		artifactStore: artifacts,
		now:           time.Now,
	}
}

func (s *ArtworkService) CreateDraft(req model.CreateArtworkDraftRequest) (model.ArtworkDraft, error) {
	if strings.TrimSpace(req.OwnerUserID) == "" {
		return model.ArtworkDraft{}, errors.New("owner_user_id is required")
	}
	layout, err := s.layout.Preview(req.Layout)
	if err != nil {
		return model.ArtworkDraft{}, err
	}

	now := s.now().UTC().Format(time.RFC3339)
	draft := model.ArtworkDraft{
		OwnerUserID:    req.OwnerUserID,
		Text:           req.Layout.Text,
		Layout:         layout,
		GlyphOverrides: req.GlyphOverrides,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return s.store.Create(draft), nil
}

func (s *ArtworkService) GetDraft(artworkID string) (model.ArtworkDraft, bool) {
	return s.store.Get(artworkID)
}

func (s *ArtworkService) ListDrafts(ownerUserID string) []model.ArtworkDraft {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil
	}
	return s.store.ListByOwner(ownerUserID)
}

func (s *ArtworkService) DeleteDraft(artworkID string) bool {
	if strings.TrimSpace(artworkID) == "" {
		return false
	}
	return s.store.Delete(artworkID)
}

func (s *ArtworkService) ExportDraft(artworkID string, req model.CreateExportRequest) (model.ExportRecord, error) {
	draft, ok := s.store.Get(artworkID)
	if !ok {
		return model.ExportRecord{}, errors.New("artwork draft not found")
	}
	if req.Format == "" {
		req.Format = "svg"
	}
	if req.Format != "svg" {
		return model.ExportRecord{}, fmt.Errorf("unsupported export format %q", req.Format)
	}
	if req.TemplateType == "" {
		req.TemplateType = "reference"
	}

	content := s.renderer.Render(draft.Layout)
	hash := sha256.Sum256([]byte(content))
	now := s.now().UTC().Format(time.RFC3339)
	record := model.ExportRecord{
		ExportID:     fmt.Sprintf("export-%06d", len(draft.Exports)+1),
		ArtworkID:    draft.ArtworkID,
		Format:       req.Format,
		TemplateType: req.TemplateType,
		ContentType:  "image/svg+xml",
		SHA256:       hex.EncodeToString(hash[:]),
		ByteSize:     len([]byte(content)),
		CreatedAt:    now,
	}
	if s.artifactStore == nil {
		record.InlineContent = content
	} else {
		storageKey, err := s.artifactStore.Save(record, content)
		if err != nil {
			return model.ExportRecord{}, err
		}
		record.StorageKey = storageKey
	}
	draft.Exports = append(draft.Exports, record)
	draft.UpdatedAt = now
	s.store.Update(draft)
	return record, nil
}

type SVGRenderer struct{}

func NewSVGRenderer() *SVGRenderer {
	return &SVGRenderer{}
}

func (r *SVGRenderer) Render(layout model.LayoutResult) string {
	width := layout.Paper.WidthCM * 10
	height := layout.Paper.HeightCM * 10
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0fmm" height="%.0fmm" viewBox="0 0 %.0f %.0f">`, layout.Paper.WidthCM*10, layout.Paper.HeightCM*10, width, height))
	builder.WriteString(`<rect width="100%" height="100%" fill="#fbf7ef"/>`)
	builder.WriteString(`<g fill="#111" text-anchor="middle" dominant-baseline="central" font-family="serif">`)
	for _, slot := range layout.Slots {
		builder.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-size="%.2f">%s</text>`, slot.XCM*10, slot.YCM*10, slot.SizeCM*10, html.EscapeString(slot.Character)))
	}
	builder.WriteString(`</g>`)
	if len(layout.SignatureSlots) > 0 {
		builder.WriteString(`<g fill="#333" text-anchor="middle" dominant-baseline="central" font-family="serif">`)
		for _, slot := range layout.SignatureSlots {
			builder.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-size="%.2f">%s</text>`, slot.XCM*10, slot.YCM*10, slot.SizeCM*10, html.EscapeString(slot.Text)))
		}
		builder.WriteString(`</g>`)
	}
	if len(layout.SealSlots) > 0 {
		builder.WriteString(`<g fill="none" stroke="#9f1d20" stroke-width="1.5">`)
		for _, slot := range layout.SealSlots {
			size := slot.SizeCM * 10
			builder.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`, slot.XCM*10-size/2, slot.YCM*10-size/2, size, size))
		}
		builder.WriteString(`</g>`)
	}
	builder.WriteString(`</svg>`)
	return builder.String()
}
