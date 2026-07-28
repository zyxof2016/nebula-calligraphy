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
CREATE TABLE IF NOT EXISTS calligraphy_schema_migrations (
  version integer PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

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
  expires_at timestamptz NOT NULL,
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

INSERT INTO calligraphy_schema_migrations(version) VALUES(2) ON CONFLICT(version) DO NOTHING;
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
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(PostgresMigrationSQL); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
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

func (s *PostgresAuthStore) FindUserByUsername(username string) (storedUser, bool, error) {
	return s.findUser("username", username)
}

func (s *PostgresAuthStore) FindUserByID(userID string) (storedUser, bool, error) {
	return s.findUser("user_id", userID)
}

func (s *PostgresAuthStore) findUser(column, value string) (storedUser, bool, error) {
	query := fmt.Sprintf("SELECT user_id, username, password_hash, created_at::text FROM calligraphy_auth_users WHERE %s=$1", column)
	var user storedUser
	if err := s.db.QueryRow(query, value).Scan(&user.UserID, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedUser{}, false, nil
		}
		return storedUser{}, false, err
	}
	return user, true, nil
}

func (s *PostgresAuthStore) SaveSession(token string, session storedSession) error {
	_, err := s.db.Exec(
		`INSERT INTO calligraphy_auth_sessions(token, user_id, expires_at)
		 VALUES($1,$2,$3)
		 ON CONFLICT(token) DO UPDATE SET user_id=excluded.user_id, expires_at=excluded.expires_at`,
		token, session.UserID, session.ExpiresAt,
	)
	return err
}

func (s *PostgresAuthStore) FindSession(token string) (storedSession, bool, error) {
	var session storedSession
	if err := s.db.QueryRow(
		"SELECT user_id, expires_at::text FROM calligraphy_auth_sessions WHERE token=$1",
		token,
	).Scan(&session.UserID, &session.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedSession{}, false, nil
		}
		return storedSession{}, false, err
	}
	return session, true, nil
}

func (s *PostgresAuthStore) DeleteSession(token string) (bool, error) {
	result, err := s.db.Exec("DELETE FROM calligraphy_auth_sessions WHERE token=$1", token)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

type PostgresArtworkStore struct {
	db *sql.DB
}

func NewPostgresArtworkStore(db *sql.DB) *PostgresArtworkStore {
	return &PostgresArtworkStore{db: db}
}

func (s *PostgresArtworkStore) Create(draft model.ArtworkDraft) (model.ArtworkDraft, error) {
	if draft.ArtworkID == "" {
		if err := s.db.QueryRow("SELECT 'artwork-' || lpad(nextval('calligraphy_artwork_seq')::text, 6, '0')").Scan(&draft.ArtworkID); err != nil {
			return model.ArtworkDraft{}, err
		}
	}
	if err := s.upsert(draft); err != nil {
		return model.ArtworkDraft{}, err
	}
	return draft, nil
}

func (s *PostgresArtworkStore) Get(artworkID string) (model.ArtworkDraft, bool, error) {
	var payload []byte
	if err := s.db.QueryRow("SELECT payload FROM calligraphy_artwork_drafts WHERE artwork_id=$1", artworkID).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ArtworkDraft{}, false, nil
		}
		return model.ArtworkDraft{}, false, err
	}
	var draft model.ArtworkDraft
	if err := json.Unmarshal(payload, &draft); err != nil {
		return model.ArtworkDraft{}, false, err
	}
	return draft, true, nil
}

func (s *PostgresArtworkStore) ListByOwner(ownerUserID string) ([]model.ArtworkDraft, error) {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_artwork_drafts WHERE owner_user_id=$1 ORDER BY created_at", ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.ArtworkDraft, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var draft model.ArtworkDraft
		if err := json.Unmarshal(payload, &draft); err != nil {
			return nil, err
		}
		items = append(items, draft)
	}
	return items, rows.Err()
}

func (s *PostgresArtworkStore) Update(draft model.ArtworkDraft) error {
	return s.upsert(draft)
}

func (s *PostgresArtworkStore) Delete(artworkID string) (bool, error) {
	result, err := s.db.Exec("DELETE FROM calligraphy_artwork_drafts WHERE artwork_id=$1", artworkID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
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

func (s *PostgresLearningStore) SaveFavorite(favorite model.FavoriteGlyph) (model.FavoriteGlyph, error) {
	payload, err := json.Marshal(favorite)
	if err != nil {
		return model.FavoriteGlyph{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO calligraphy_learning_favorites(owner_user_id, glyph_id, payload, created_at)
		 VALUES($1,$2,$3,$4)
		 ON CONFLICT(owner_user_id, glyph_id) DO UPDATE SET payload=excluded.payload, created_at=excluded.created_at`,
		favorite.OwnerUserID, favorite.GlyphID, payload, favorite.CreatedAt,
	)
	return favorite, err
}

func (s *PostgresLearningStore) DeleteFavorite(ownerUserID, glyphID string) (bool, error) {
	result, err := s.db.Exec("DELETE FROM calligraphy_learning_favorites WHERE owner_user_id=$1 AND glyph_id=$2", ownerUserID, glyphID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (s *PostgresLearningStore) ListFavorites(ownerUserID string) ([]model.FavoriteGlyph, error) {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_learning_favorites WHERE owner_user_id=$1 ORDER BY created_at DESC", ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FavoriteGlyph, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var favorite model.FavoriteGlyph
		if err := json.Unmarshal(payload, &favorite); err != nil {
			return nil, err
		}
		items = append(items, favorite)
	}
	return items, rows.Err()
}

func (s *PostgresLearningStore) AddPractice(record model.PracticeRecord) (model.PracticeRecord, error) {
	if record.PracticeID == "" {
		if err := s.db.QueryRow("SELECT 'practice-' || lpad(nextval('calligraphy_practice_seq')::text, 6, '0')").Scan(&record.PracticeID); err != nil {
			return model.PracticeRecord{}, err
		}
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return model.PracticeRecord{}, err
	}
	_, err = s.db.Exec(
		"INSERT INTO calligraphy_learning_practice(practice_id, owner_user_id, payload, created_at) VALUES($1,$2,$3,$4)",
		record.PracticeID, record.OwnerUserID, payload, record.CreatedAt,
	)
	return record, err
}

func (s *PostgresLearningStore) ListPractice(ownerUserID string) ([]model.PracticeRecord, error) {
	rows, err := s.db.Query("SELECT payload FROM calligraphy_learning_practice WHERE owner_user_id=$1 ORDER BY created_at DESC LIMIT 20", ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.PracticeRecord, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var record model.PracticeRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, rows.Err()
}
