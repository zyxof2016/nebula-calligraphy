package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

const PostgresMigrationSQL = `
CREATE SEQUENCE IF NOT EXISTS calligraphy_user_seq;
CREATE SEQUENCE IF NOT EXISTS calligraphy_artwork_seq;
CREATE SEQUENCE IF NOT EXISTS calligraphy_practice_seq;

CREATE TABLE IF NOT EXISTS calligraphy_auth_users (
  user_id text PRIMARY KEY,
  username text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS calligraphy_auth_sessions (
  token text PRIMARY KEY,
  user_id text NOT NULL REFERENCES calligraphy_auth_users(user_id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS calligraphy_artwork_drafts (
  artwork_id text PRIMARY KEY,
  owner_user_id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_calligraphy_artwork_owner ON calligraphy_artwork_drafts(owner_user_id);

CREATE TABLE IF NOT EXISTS calligraphy_learning_favorites (
  owner_user_id text NOT NULL,
  glyph_id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY(owner_user_id, glyph_id)
);

CREATE TABLE IF NOT EXISTS calligraphy_learning_practice (
  practice_id text PRIMARY KEY,
  owner_user_id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_calligraphy_practice_owner ON calligraphy_learning_practice(owner_user_id);
`

func OpenPostgres(databaseURL string) (*sql.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("postgres database url is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func MigratePostgres(db *sql.DB) error {
	_, err := db.Exec(PostgresMigrationSQL)
	return err
}

type PostgresAuthStore struct {
	db *sql.DB
}

func NewPostgresAuthStore(db *sql.DB) *PostgresAuthStore {
	return &PostgresAuthStore{db: db}
}

func (s *PostgresAuthStore) CreateUser(user storedUser) (storedUser, error) {
	if user.UserID == "" {
		if err := s.db.QueryRow("SELECT 'user-' || lpad(nextval('calligraphy_user_seq')::text, 6, '0')").Scan(&user.UserID); err != nil {
			return storedUser{}, err
		}
	}
	_, err := s.db.Exec(
		"INSERT INTO calligraphy_auth_users(user_id, username, password_hash, created_at) VALUES($1,$2,$3,$4)",
		user.UserID, user.Username, user.PasswordHash, user.CreatedAt,
	)
	return user, err
}

func (s *PostgresAuthStore) FindUserByUsername(username string) (storedUser, bool) {
	return s.findUser("username", username)
}

func (s *PostgresAuthStore) FindUserByID(userID string) (storedUser, bool) {
	return s.findUser("user_id", userID)
}

func (s *PostgresAuthStore) findUser(column, value string) (storedUser, bool) {
	query := fmt.Sprintf("SELECT user_id, username, password_hash, created_at::text FROM calligraphy_auth_users WHERE %s=$1", column)
	var user storedUser
	if err := s.db.QueryRow(query, value).Scan(&user.UserID, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		return storedUser{}, false
	}
	return user, true
}

func (s *PostgresAuthStore) SaveSession(token, userID string) {
	_, _ = s.db.Exec("INSERT INTO calligraphy_auth_sessions(token, user_id) VALUES($1,$2) ON CONFLICT(token) DO UPDATE SET user_id=excluded.user_id", token, userID)
}

func (s *PostgresAuthStore) FindSession(token string) (string, bool) {
	var userID string
	if err := s.db.QueryRow("SELECT user_id FROM calligraphy_auth_sessions WHERE token=$1", token).Scan(&userID); err != nil {
		return "", false
	}
	return userID, true
}

func (s *PostgresAuthStore) DeleteSession(token string) bool {
	result, err := s.db.Exec("DELETE FROM calligraphy_auth_sessions WHERE token=$1", token)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

type PostgresArtworkStore struct {
	db *sql.DB
}

func NewPostgresArtworkStore(db *sql.DB) *PostgresArtworkStore {
	return &PostgresArtworkStore{db: db}
}

func (s *PostgresArtworkStore) Create(draft model.ArtworkDraft) model.ArtworkDraft {
	if draft.ArtworkID == "" {
		_ = s.db.QueryRow("SELECT 'artwork-' || lpad(nextval('calligraphy_artwork_seq')::text, 6, '0')").Scan(&draft.ArtworkID)
	}
	_ = s.upsert(draft)
	return draft
}

func (s *PostgresArtworkStore) Get(artworkID string) (model.ArtworkDraft, bool) {
	var payload []byte
	if err := s.db.QueryRow("SELECT payload FROM calligraphy_artwork_drafts WHERE artwork_id=$1", artworkID).Scan(&payload); err != nil {
		return model.ArtworkDraft{}, false
	}
	var draft model.ArtworkDraft
	if err := json.Unmarshal(payload, &draft); err != nil {
		return model.ArtworkDraft{}, false
	}
	return draft, true
}

func (s *PostgresArtworkStore) ListByOwner(ownerUserID string) []model.ArtworkDraft {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_artwork_drafts WHERE owner_user_id=$1 ORDER BY created_at", ownerUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]model.ArtworkDraft, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			continue
		}
		var draft model.ArtworkDraft
		if err := json.Unmarshal(payload, &draft); err == nil {
			items = append(items, draft)
		}
	}
	return items
}

func (s *PostgresArtworkStore) Update(draft model.ArtworkDraft) model.ArtworkDraft {
	_ = s.upsert(draft)
	return draft
}

func (s *PostgresArtworkStore) Delete(artworkID string) bool {
	result, err := s.db.Exec("DELETE FROM calligraphy_artwork_drafts WHERE artwork_id=$1", artworkID)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *PostgresArtworkStore) upsert(draft model.ArtworkDraft) error {
	payload, err := json.Marshal(draft)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO calligraphy_artwork_drafts(artwork_id, owner_user_id, payload, created_at, updated_at)
		 VALUES($1,$2,$3,$4,$5)
		 ON CONFLICT(artwork_id) DO UPDATE SET payload=excluded.payload, updated_at=excluded.updated_at`,
		draft.ArtworkID, draft.OwnerUserID, payload, draft.CreatedAt, draft.UpdatedAt,
	)
	return err
}

type PostgresLearningStore struct {
	db *sql.DB
}

func NewPostgresLearningStore(db *sql.DB) *PostgresLearningStore {
	return &PostgresLearningStore{db: db}
}

func (s *PostgresLearningStore) SaveFavorite(favorite model.FavoriteGlyph) model.FavoriteGlyph {
	payload, _ := json.Marshal(favorite)
	_, _ = s.db.Exec(
		`INSERT INTO calligraphy_learning_favorites(owner_user_id, glyph_id, payload, created_at)
		 VALUES($1,$2,$3,$4)
		 ON CONFLICT(owner_user_id, glyph_id) DO UPDATE SET payload=excluded.payload, created_at=excluded.created_at`,
		favorite.OwnerUserID, favorite.GlyphID, payload, favorite.CreatedAt,
	)
	return favorite
}

func (s *PostgresLearningStore) DeleteFavorite(ownerUserID, glyphID string) bool {
	result, err := s.db.Exec("DELETE FROM calligraphy_learning_favorites WHERE owner_user_id=$1 AND glyph_id=$2", ownerUserID, glyphID)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *PostgresLearningStore) ListFavorites(ownerUserID string) []model.FavoriteGlyph {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_learning_favorites WHERE owner_user_id=$1 ORDER BY created_at DESC", ownerUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]model.FavoriteGlyph, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			continue
		}
		var favorite model.FavoriteGlyph
		if err := json.Unmarshal(payload, &favorite); err == nil {
			items = append(items, favorite)
		}
	}
	return items
}

func (s *PostgresLearningStore) AddPractice(record model.PracticeRecord) model.PracticeRecord {
	if record.PracticeID == "" {
		_ = s.db.QueryRow("SELECT 'practice-' || lpad(nextval('calligraphy_practice_seq')::text, 6, '0')").Scan(&record.PracticeID)
	}
	payload, _ := json.Marshal(record)
	_, _ = s.db.Exec(
		"INSERT INTO calligraphy_learning_practice(practice_id, owner_user_id, payload, created_at) VALUES($1,$2,$3,$4)",
		record.PracticeID, record.OwnerUserID, payload, record.CreatedAt,
	)
	return record
}

func (s *PostgresLearningStore) ListPractice(ownerUserID string) []model.PracticeRecord {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_learning_practice WHERE owner_user_id=$1 ORDER BY created_at DESC LIMIT 20", ownerUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]model.PracticeRecord, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			continue
		}
		var record model.PracticeRecord
		if err := json.Unmarshal(payload, &record); err == nil {
			items = append(items, record)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items
}
