package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"golang.org/x/crypto/argon2"
)

const (
	argonTime         = 3
	argonMemory       = 64 * 1024
	argonThreads      = 2
	argonKeyLength    = 32
	maxLoginFailures  = 5
	defaultSessionTTL = 24 * time.Hour
)

var loginLockDuration = 15 * time.Minute

type AuthStore interface {
	CreateUser(user storedUser) (storedUser, error)
	FindUserByUsername(username string) (storedUser, bool, error)
	FindUserByID(userID string) (storedUser, bool, error)
	SaveSession(token string, session storedSession) error
	FindSession(token string) (storedSession, bool, error)
	DeleteSession(token string) (bool, error)
}

type storedUser struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	CreatedAt    string `json:"created_at"`
}

type storedSession struct {
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

type authState struct {
	NextUser int                      `json:"next_user"`
	Users    map[string]storedUser    `json:"users"`
	Sessions map[string]storedSession `json:"sessions"`
}

type InMemoryAuthStore struct {
	mu       sync.RWMutex
	nextUser int
	users    map[string]storedUser
	sessions map[string]storedSession
}

func NewInMemoryAuthStore() *InMemoryAuthStore {
	return &InMemoryAuthStore{
		users:    make(map[string]storedUser),
		sessions: make(map[string]storedSession),
	}
}

func (s *InMemoryAuthStore) CreateUser(user storedUser) (storedUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[user.Username]; ok {
		return storedUser{}, errors.New("username already exists")
	}
	s.nextUser++
	user.UserID = fmt.Sprintf("user-%06d", s.nextUser)
	s.users[user.Username] = user
	return user, nil
}

func (s *InMemoryAuthStore) FindUserByUsername(username string) (storedUser, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	return user, ok, nil
}

func (s *InMemoryAuthStore) FindUserByID(userID string) (storedUser, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.UserID == userID {
			return user, true, nil
		}
	}
	return storedUser{}, false, nil
}

func (s *InMemoryAuthStore) SaveSession(token string, session storedSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[token] = session
	return nil
}

func (s *InMemoryAuthStore) FindSession(token string) (storedSession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	return session, ok, nil
}

func (s *InMemoryAuthStore) DeleteSession(token string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[token]; !ok {
		return false, nil
	}
	delete(s.sessions, token)
	return true, nil
}

type FileAuthStore struct {
	mu       sync.RWMutex
	path     string
	nextUser int
	users    map[string]storedUser
	sessions map[string]storedSession
}

func NewFileAuthStore(path string) (*FileAuthStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("auth store path is required")
	}
	store := &FileAuthStore{
		path:     path,
		users:    make(map[string]storedUser),
		sessions: make(map[string]storedSession),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileAuthStore) CreateUser(user storedUser) (storedUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[user.Username]; ok {
		return storedUser{}, errors.New("username already exists")
	}
	previousNext := s.nextUser
	s.nextUser++
	user.UserID = fmt.Sprintf("user-%06d", s.nextUser)
	s.users[user.Username] = user
	if err := s.persistLocked(); err != nil {
		delete(s.users, user.Username)
		s.nextUser = previousNext
		return storedUser{}, err
	}
	return user, nil
}

func (s *FileAuthStore) FindUserByUsername(username string) (storedUser, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	return user, ok, nil
}

func (s *FileAuthStore) FindUserByID(userID string) (storedUser, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.UserID == userID {
			return user, true, nil
		}
	}
	return storedUser{}, false, nil
}

func (s *FileAuthStore) SaveSession(token string, session storedSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous, existed := s.sessions[token]
	s.sessions[token] = session
	if err := s.persistLocked(); err != nil {
		if existed {
			s.sessions[token] = previous
		} else {
			delete(s.sessions, token)
		}
		return err
	}
	return nil
}

func (s *FileAuthStore) FindSession(token string) (storedSession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	return session, ok, nil
}

func (s *FileAuthStore) DeleteSession(token string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous, ok := s.sessions[token]
	if !ok {
		return false, nil
	}
	delete(s.sessions, token)
	if err := s.persistLocked(); err != nil {
		s.sessions[token] = previous
		return false, err
	}
	return true, nil
}

func (s *FileAuthStore) load() error {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var state authState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}
	s.nextUser = state.NextUser
	if state.Users != nil {
		s.users = state.Users
	}
	if state.Sessions != nil {
		s.sessions = state.Sessions
	}
	return nil
}

func (s *FileAuthStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(authState{NextUser: s.nextUser, Users: s.users, Sessions: s.sessions}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

type AuthService struct {
	store         AuthStore
	now           func() time.Time
	tokenSource   func() (string, error)
	saltSource    func() (string, error)
	guardMu       sync.Mutex
	loginFailures map[string]loginFailureState
	sessionTTL    time.Duration
}

type loginFailureState struct {
	Count       int
	LockedUntil time.Time
}

func NewAuthService(store AuthStore) *AuthService {
	return &AuthService{
		store:         store,
		now:           time.Now,
		tokenSource:   randomHex,
		saltSource:    randomHex,
		loginFailures: make(map[string]loginFailureState),
		sessionTTL:    defaultSessionTTL,
	}
}

func (s *AuthService) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		s.sessionTTL = ttl
	}
}

func (s *AuthService) Register(req model.AuthRequest) (model.AuthSession, error) {
	username, password, err := normalizeAuthRequest(req)
	if err != nil {
		return model.AuthSession{}, err
	}
	if _, ok, err := s.store.FindUserByUsername(username); err != nil {
		return model.AuthSession{}, fmt.Errorf("%w: find user for registration: %v", ErrPersistence, err)
	} else if ok {
		return model.AuthSession{}, errors.New("username already exists")
	}
	salt, err := s.saltSource()
	if err != nil {
		return model.AuthSession{}, err
	}
	user, err := s.store.CreateUser(storedUser{
		Username:     username,
		PasswordHash: encodePassword(password, salt),
		CreatedAt:    s.now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return model.AuthSession{}, fmt.Errorf("%w: create user: %v", ErrPersistence, err)
	}
	return s.createSession(user)
}

func (s *AuthService) Login(req model.AuthRequest) (model.AuthSession, error) {
	username, password, err := normalizeAuthRequest(req)
	if err != nil {
		return model.AuthSession{}, err
	}
	if s.isLoginLocked(username) {
		return model.AuthSession{}, errors.New("login temporarily locked")
	}
	user, ok, err := s.store.FindUserByUsername(username)
	if err != nil {
		return model.AuthSession{}, fmt.Errorf("%w: find user for login: %v", ErrPersistence, err)
	}
	if !ok || !verifyPassword(password, user.PasswordHash) {
		s.recordLoginFailure(username)
		return model.AuthSession{}, errors.New("invalid username or password")
	}
	s.clearLoginFailures(username)
	return s.createSession(user)
}

func (s *AuthService) CurrentUser(token string) (model.User, error) {
	token = strings.TrimSpace(token)
	session, ok, err := s.store.FindSession(token)
	if err != nil {
		return model.User{}, fmt.Errorf("%w: find auth session: %v", ErrPersistence, err)
	}
	if !ok {
		return model.User{}, ErrUnauthorized
	}
	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || !s.now().Before(expiresAt) {
		_, _ = s.store.DeleteSession(token)
		return model.User{}, ErrUnauthorized
	}
	user, ok, err := s.store.FindUserByID(session.UserID)
	if err != nil {
		return model.User{}, fmt.Errorf("%w: find current user: %v", ErrPersistence, err)
	}
	if !ok {
		return model.User{}, ErrUnauthorized
	}
	return publicUser(user), nil
}

func (s *AuthService) Logout(token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	return s.store.DeleteSession(token)
}

func (s *AuthService) createSession(user storedUser) (model.AuthSession, error) {
	token, err := s.tokenSource()
	if err != nil {
		return model.AuthSession{}, err
	}
	expiresAt := s.now().Add(s.sessionTTL).UTC().Format(time.RFC3339)
	if err := s.store.SaveSession(token, storedSession{UserID: user.UserID, ExpiresAt: expiresAt}); err != nil {
		return model.AuthSession{}, fmt.Errorf("%w: save auth session: %v", ErrPersistence, err)
	}
	return model.AuthSession{Token: token, ExpiresAt: expiresAt, User: publicUser(user)}, nil
}

func (s *AuthService) isLoginLocked(username string) bool {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()

	state := s.loginFailures[username]
	if state.LockedUntil.IsZero() {
		return false
	}
	if s.now().Before(state.LockedUntil) {
		return true
	}
	delete(s.loginFailures, username)
	return false
}

func (s *AuthService) recordLoginFailure(username string) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()

	state := s.loginFailures[username]
	state.Count++
	if state.Count >= maxLoginFailures {
		state.LockedUntil = s.now().Add(loginLockDuration)
	}
	s.loginFailures[username] = state
}

func (s *AuthService) clearLoginFailures(username string) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()

	delete(s.loginFailures, username)
}

func normalizeAuthRequest(req model.AuthRequest) (string, string, error) {
	username := strings.ToLower(strings.TrimSpace(req.Username))
	password := strings.TrimSpace(req.Password)
	if len(username) < 3 {
		return "", "", errors.New("username must be at least 3 characters")
	}
	if len(password) < 8 {
		return "", "", errors.New("password must be at least 8 characters")
	}
	return username, password, nil
}

func publicUser(user storedUser) model.User {
	return model.User{UserID: user.UserID, Username: user.Username, CreatedAt: user.CreatedAt}
}

func randomHex() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func encodePassword(password, salt string) string {
	hash := argon2.IDKey([]byte(password), []byte(salt), argonTime, argonMemory, argonThreads, argonKeyLength)
	return fmt.Sprintf("argon2id:%d:%d:%d:%s:%s", argonTime, argonMemory, argonThreads, salt, hex.EncodeToString(hash))
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, ":")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return false
	}
	timeCost, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return false
	}
	memoryCost, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return false
	}
	threads, err := strconv.ParseUint(parts[3], 10, 8)
	if err != nil || threads == 0 {
		return false
	}
	expected, err := hex.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), []byte(parts[4]), uint32(timeCost), uint32(memoryCost), uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
