package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type LearningStore interface {
	SaveFavorite(favorite model.FavoriteGlyph) (model.FavoriteGlyph, error)
	DeleteFavorite(ownerUserID, glyphID string) (bool, error)
	ListFavorites(ownerUserID string) ([]model.FavoriteGlyph, error)
	AddPractice(record model.PracticeRecord) (model.PracticeRecord, error)
	ListPractice(ownerUserID string) ([]model.PracticeRecord, error)
}

type learningState struct {
	NextPractice int                            `json:"next_practice"`
	Favorites    map[string]model.FavoriteGlyph `json:"favorites"`
	Practice     []model.PracticeRecord         `json:"practice"`
}

type InMemoryLearningStore struct {
	mu           sync.RWMutex
	nextPractice int
	favorites    map[string]model.FavoriteGlyph
	practice     []model.PracticeRecord
}

func NewInMemoryLearningStore() *InMemoryLearningStore {
	return &InMemoryLearningStore{
		favorites: make(map[string]model.FavoriteGlyph),
		practice:  make([]model.PracticeRecord, 0),
	}
}

func (s *InMemoryLearningStore) SaveFavorite(favorite model.FavoriteGlyph) (model.FavoriteGlyph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.favorites[favoriteKey(favorite.OwnerUserID, favorite.GlyphID)] = favorite
	return favorite, nil
}

func (s *InMemoryLearningStore) DeleteFavorite(ownerUserID, glyphID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := favoriteKey(ownerUserID, glyphID)
	if _, ok := s.favorites[key]; !ok {
		return false, nil
	}
	delete(s.favorites, key)
	return true, nil
}

func (s *InMemoryLearningStore) ListFavorites(ownerUserID string) ([]model.FavoriteGlyph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.FavoriteGlyph, 0)
	for _, favorite := range s.favorites {
		if favorite.OwnerUserID == ownerUserID {
			items = append(items, favorite)
		}
	}
	sortFavorites(items)
	return items, nil
}

func (s *InMemoryLearningStore) AddPractice(record model.PracticeRecord) (model.PracticeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextPractice++
	record.PracticeID = fmt.Sprintf("practice-%06d", s.nextPractice)
	s.practice = append(s.practice, record)
	return record, nil
}

func (s *InMemoryLearningStore) ListPractice(ownerUserID string) ([]model.PracticeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return filterPractice(s.practice, ownerUserID), nil
}

type FileLearningStore struct {
	mu           sync.RWMutex
	path         string
	nextPractice int
	favorites    map[string]model.FavoriteGlyph
	practice     []model.PracticeRecord
}

func NewFileLearningStore(path string) (*FileLearningStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("learning store path is required")
	}
	store := &FileLearningStore{
		path:      path,
		favorites: make(map[string]model.FavoriteGlyph),
		practice:  make([]model.PracticeRecord, 0),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileLearningStore) SaveFavorite(favorite model.FavoriteGlyph) (model.FavoriteGlyph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := favoriteKey(favorite.OwnerUserID, favorite.GlyphID)
	previous, existed := s.favorites[key]
	s.favorites[key] = favorite
	if err := s.persistLocked(); err != nil {
		if existed {
			s.favorites[key] = previous
		} else {
			delete(s.favorites, key)
		}
		return model.FavoriteGlyph{}, err
	}
	return favorite, nil
}

func (s *FileLearningStore) DeleteFavorite(ownerUserID, glyphID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := favoriteKey(ownerUserID, glyphID)
	previous, ok := s.favorites[key]
	if !ok {
		return false, nil
	}
	delete(s.favorites, key)
	if err := s.persistLocked(); err != nil {
		s.favorites[key] = previous
		return false, err
	}
	return true, nil
}

func (s *FileLearningStore) ListFavorites(ownerUserID string) ([]model.FavoriteGlyph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.FavoriteGlyph, 0)
	for _, favorite := range s.favorites {
		if favorite.OwnerUserID == ownerUserID {
			items = append(items, favorite)
		}
	}
	sortFavorites(items)
	return items, nil
}

func (s *FileLearningStore) AddPractice(record model.PracticeRecord) (model.PracticeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previousNext := s.nextPractice
	s.nextPractice++
	record.PracticeID = fmt.Sprintf("practice-%06d", s.nextPractice)
	s.practice = append(s.practice, record)
	if err := s.persistLocked(); err != nil {
		s.practice = s.practice[:len(s.practice)-1]
		s.nextPractice = previousNext
		return model.PracticeRecord{}, err
	}
	return record, nil
}

func (s *FileLearningStore) ListPractice(ownerUserID string) ([]model.PracticeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return filterPractice(s.practice, ownerUserID), nil
}

func (s *FileLearningStore) load() error {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var state learningState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}
	s.nextPractice = state.NextPractice
	if state.Favorites != nil {
		s.favorites = state.Favorites
	}
	if state.Practice != nil {
		s.practice = state.Practice
	}
	return nil
}

func (s *FileLearningStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(learningState{
		NextPractice: s.nextPractice,
		Favorites:    s.favorites,
		Practice:     s.practice,
	}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

type LearningService struct {
	store    LearningStore
	catalog  GlyphCatalog
	now      func() time.Time
	location *time.Location
}

func NewLearningService(store LearningStore, catalog GlyphCatalog) *LearningService {
	return &LearningService{store: store, catalog: catalog, now: time.Now, location: defaultLearningLocation()}
}

func (s *LearningService) SetLearningLocation(location *time.Location) {
	if location != nil {
		s.location = location
	}
}

func defaultLearningLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return location
}

func (s *LearningService) AddFavorite(ownerUserID string, req model.CreateFavoriteRequest) (model.FavoriteGlyph, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return model.FavoriteGlyph{}, errors.New("owner_user_id is required")
	}
	detail, err := s.findGlyph(req.GlyphID)
	if err != nil {
		return model.FavoriteGlyph{}, err
	}
	now := s.now().UTC().Format(time.RFC3339)
	favorite := model.FavoriteGlyph{
		OwnerUserID: ownerUserID,
		GlyphID:     detail.Glyph.GlyphID,
		Character:   detail.Glyph.Character,
		Style:       detail.Glyph.Style,
		CopybookID:  detail.Glyph.CopybookID,
		CreatedAt:   now,
	}
	saved, err := s.store.SaveFavorite(favorite)
	if err != nil {
		return model.FavoriteGlyph{}, fmt.Errorf("%w: save favorite: %v", ErrPersistence, err)
	}
	return saved, nil
}

func (s *LearningService) DeleteFavorite(ownerUserID, glyphID string) (bool, error) {
	if strings.TrimSpace(ownerUserID) == "" || strings.TrimSpace(glyphID) == "" {
		return false, errors.New("owner_user_id and glyph_id are required")
	}
	deleted, err := s.store.DeleteFavorite(ownerUserID, glyphID)
	if err != nil {
		return false, fmt.Errorf("%w: delete favorite: %v", ErrPersistence, err)
	}
	return deleted, nil
}

func (s *LearningService) RecordPractice(ownerUserID string, req model.CreatePracticeRecordRequest) (model.PracticeRecord, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return model.PracticeRecord{}, errors.New("owner_user_id is required")
	}
	detail, err := s.findGlyph(req.GlyphID)
	if err != nil {
		return model.PracticeRecord{}, err
	}
	if strings.TrimSpace(req.TemplateType) == "" {
		req.TemplateType = "copy"
	}
	if strings.TrimSpace(req.GridType) == "" {
		req.GridType = "mi"
	}
	record := model.PracticeRecord{
		OwnerUserID:  ownerUserID,
		GlyphID:      detail.Glyph.GlyphID,
		Character:    detail.Glyph.Character,
		Style:        detail.Glyph.Style,
		TemplateType: req.TemplateType,
		GridType:     req.GridType,
		CreatedAt:    s.now().UTC().Format(time.RFC3339),
	}
	saved, err := s.store.AddPractice(record)
	if err != nil {
		return model.PracticeRecord{}, fmt.Errorf("%w: save practice record: %v", ErrPersistence, err)
	}
	return saved, nil
}

func (s *LearningService) GetProfile(ownerUserID string) (model.LearningProfile, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return model.LearningProfile{}, errors.New("owner_user_id is required")
	}
	favorites, err := s.store.ListFavorites(ownerUserID)
	if err != nil {
		return model.LearningProfile{}, fmt.Errorf("%w: list favorites: %v", ErrPersistence, err)
	}
	practice, err := s.store.ListPractice(ownerUserID)
	if err != nil {
		return model.LearningProfile{}, fmt.Errorf("%w: list practice records: %v", ErrPersistence, err)
	}
	todayPractice := filterPracticeByDate(practice, s.now(), s.location)
	dailyPlan := s.buildDailyPlan(favorites, practice)
	profile := model.LearningProfile{
		OwnerUserID:        ownerUserID,
		Favorites:          favorites,
		RecentPractice:     practice,
		DailyPlan:          dailyPlan,
		DailySteps:         buildDailySteps(dailyPlan, todayPractice),
		PracticeCount:      len(practice),
		TodayPracticeCount: len(todayPractice),
		FavoriteCount:      len(favorites),
	}
	if len(practice) > 0 {
		profile.LastPracticedAt = practice[0].CreatedAt
	}
	return profile, nil
}

func filterPracticeByDate(practice []model.PracticeRecord, day time.Time, location *time.Location) []model.PracticeRecord {
	if location == nil {
		location = time.UTC
	}
	year, month, date := day.In(location).Date()
	items := make([]model.PracticeRecord, 0, len(practice))
	for _, record := range practice {
		createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
		if err != nil {
			continue
		}
		recordYear, recordMonth, recordDate := createdAt.In(location).Date()
		if recordYear == year && recordMonth == month && recordDate == date {
			items = append(items, record)
		}
	}
	return items
}

func buildDailySteps(plan []model.PracticeSuggestion, todayPractice []model.PracticeRecord) []model.DailyPracticeStep {
	targetGlyphID := ""
	targetCharacter := ""
	if len(todayPractice) > 0 {
		targetGlyphID = todayPractice[0].GlyphID
		targetCharacter = todayPractice[0].Character
	} else if len(plan) > 0 {
		targetGlyphID = plan[0].GlyphID
		targetCharacter = plan[0].Character
	}
	return []model.DailyPracticeStep{
		{
			StepID:          "copy_reference",
			Title:           "临摹今日字",
			Description:     "先看参考字形，再写 3 遍。",
			ActionLabel:     "开始临摹",
			TargetGlyphID:   targetGlyphID,
			TargetCharacter: targetCharacter,
			Completed:       len(todayPractice) > 0,
		},
		{
			StepID:          "inspect_structure",
			Title:           "拆结构",
			Description:     "看中宫、重心和主笔方向。",
			ActionLabel:     "看结构",
			TargetGlyphID:   targetGlyphID,
			TargetCharacter: targetCharacter,
		},
		{
			StepID:          "switch_grid",
			Title:           "换格再写",
			Description:     "从米字格切到九宫格，检查比例。",
			ActionLabel:     "换格练",
			TargetGlyphID:   targetGlyphID,
			TargetCharacter: targetCharacter,
		},
		{
			StepID:      "compose_phrase",
			Title:       "入章法",
			Description: "把今日字放进四字短句或斗方。",
			ActionLabel: "去创作",
		},
	}
}

func (s *LearningService) buildDailyPlan(favorites []model.FavoriteGlyph, practice []model.PracticeRecord) []model.PracticeSuggestion {
	style := "ou"
	if len(practice) > 0 && strings.TrimSpace(practice[0].Style) != "" {
		style = practice[0].Style
	} else if len(favorites) > 0 && strings.TrimSpace(favorites[0].Style) != "" {
		style = favorites[0].Style
	}
	practiced := map[string]bool{}
	for _, record := range practice {
		practiced[record.GlyphID] = true
		practiced[record.Character] = true
	}
	plan := make([]model.PracticeSuggestion, 0, 5)
	for _, group := range s.catalog.ListPresetGroups(style) {
		for _, glyph := range group.Glyphs {
			if practiced[glyph.GlyphID] || practiced[glyph.Character] {
				continue
			}
			plan = append(plan, model.PracticeSuggestion{
				GlyphID:    glyph.GlyphID,
				Character:  glyph.Character,
				Style:      glyph.Style,
				CopybookID: glyph.CopybookID,
				Title:      group.Title,
				Reason:     dailyPlanReason(group),
			})
			if len(plan) >= 5 {
				return plan
			}
		}
	}
	return plan
}

func dailyPlanReason(group model.GlyphPresetGroup) string {
	switch group.GroupID {
	case "basic-strokes":
		return "巩固基础笔画和起收笔"
	case "structure":
		return "观察中宫、重心和开合"
	case "nature":
		return "为诗词创作积累常用意象字"
	case "poetry":
		return "提升唐诗高频字熟练度"
	case "virtue":
		return "适合书斋小品和斗方练习"
	default:
		if strings.TrimSpace(group.Description) != "" {
			return group.Description
		}
		return "保持每日临摹节奏"
	}
}

func (s *LearningService) findGlyph(glyphID string) (model.GlyphDetail, error) {
	glyphID = strings.TrimSpace(glyphID)
	if glyphID == "" {
		return model.GlyphDetail{}, errors.New("glyph_id is required")
	}
	detail, ok := s.catalog.GetDetail(glyphID)
	if !ok {
		return model.GlyphDetail{}, errors.New("glyph not found")
	}
	return detail, nil
}

func favoriteKey(ownerUserID, glyphID string) string {
	return ownerUserID + "\x00" + glyphID
}

func sortFavorites(items []model.FavoriteGlyph) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].GlyphID < items[j].GlyphID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}

func filterPractice(items []model.PracticeRecord, ownerUserID string) []model.PracticeRecord {
	filtered := make([]model.PracticeRecord, 0)
	for _, record := range items {
		if record.OwnerUserID == ownerUserID {
			filtered = append(filtered, record)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt == filtered[j].CreatedAt {
			return filtered[i].PracticeID > filtered[j].PracticeID
		}
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})
	if len(filtered) > 20 {
		return filtered[:20]
	}
	return filtered
}
