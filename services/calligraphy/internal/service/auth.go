package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

const passwordIterations = 120000
const maxLoginFailures = 5

var loginLockDuration = 15 * time.Minute

type AuthStore interface {
	CreateUser(user storedUser) (storedUser, error)
	FindUserByUsername(username string) (storedUser, bool)
	FindUserByID(userID string) (storedUser, bool)
	SaveSession(token, userID string)
	FindSession(token string) (string, bool)
	DeleteSession(token string) bool
}

type storedUser struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	CreatedAt    string `json:"created_at"`
}

type authState struct {
	NextUser int                   `json:"next_user"`
	Users    map[string]storedUser `json:"users"`
	Sessions map[string]string     `json:"sessions"`
}

type InMemoryAuthStore struct {
	mu       sync.RWMutex
	nextUser int
	users    map[string]storedUser
	sessions map[string]string
}

func NewInMemoryAuthStore() *InMemoryAuthStore {
	return &InMemoryAuthStore{
		users:    make(map[string]storedUser),
		sessions: make(map[string]string),
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

func (s *InMemoryAuthStore) FindUserByUsername(username string) (storedUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	return user, ok
}

func (s *InMemoryAuthStore) FindUserByID(userID string) (storedUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.UserID == userID {
			return user, true
		}
	}
	return storedUser{}, false
}

func (s *InMemoryAuthStore) SaveSession(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[token] = userID
}

func (s *InMemoryAuthStore) FindSession(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.sessions[token]
	return userID, ok
}

func (s *InMemoryAuthStore) DeleteSession(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[token]; !ok {
		return false
	}
	delete(s.sessions, token)
	return true
}

type FileAuthStore struct {
	mu       sync.RWMutex
	path     string
	nextUser int
	users    map[string]storedUser
	sessions map[string]string
}

func NewFileAuthStore(path string) (*FileAuthStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("auth store path is required")
	}
	store := &FileAuthStore{
		path:     path,
		users:    make(map[string]storedUser),
		sessions: make(map[string]string),
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
	s.nextUser++
	user.UserID = fmt.Sprintf("user-%06d", s.nextUser)
	s.users[user.Username] = user
	_ = s.persistLocked()
	return user, nil
}

func (s *FileAuthStore) FindUserByUsername(username string) (storedUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	return user, ok
}

func (s *FileAuthStore) FindUserByID(userID string) (storedUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.UserID == userID {
			return user, true
		}
	}
	return storedUser{}, false
}

func (s *FileAuthStore) SaveSession(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[token] = userID
	_ = s.persistLocked()
}

func (s *FileAuthStore) FindSession(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.sessions[token]
	return userID, ok
}

func (s *FileAuthStore) DeleteSession(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[token]; !ok {
		return false
	}
	delete(s.sessions, token)
	_ = s.persistLocked()
	return true
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
	}
}

func (s *AuthService) Register(req model.AuthRequest) (model.AuthSession, error) {
	username, password, err := normalizeAuthRequest(req)
	if err != nil {
		return model.AuthSession{}, err
	}
	if _, ok := s.store.FindUserByUsername(username); ok {
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
		return model.AuthSession{}, err
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
	user, ok := s.store.FindUserByUsername(username)
	if !ok || !verifyPassword(password, user.PasswordHash) {
		s.recordLoginFailure(username)
		return model.AuthSession{}, errors.New("invalid username or password")
	}
	s.clearLoginFailures(username)
	return s.createSession(user)
}

func (s *AuthService) CurrentUser(token string) (model.User, bool) {
	userID, ok := s.store.FindSession(strings.TrimSpace(token))
	if !ok {
		return model.User{}, false
	}
	user, ok := s.store.FindUserByID(userID)
	if !ok {
		return model.User{}, false
	}
	return publicUser(user), true
}

func (s *AuthService) Logout(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	return s.store.DeleteSession(token)
}

func (s *AuthService) createSession(user storedUser) (model.AuthSession, error) {
	token, err := s.tokenSource()
	if err != nil {
		return model.AuthSession{}, err
	}
	s.store.SaveSession(token, user.UserID)
	return model.AuthSession{Token: token, User: publicUser(user)}, nil
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
	return fmt.Sprintf("sha256:%d:%s:%s", passwordIterations, salt, derivePassword(password, salt, passwordIterations))
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, ":")
	if len(parts) != 4 || parts[0] != "sha256" {
		return false
	}
	expected := encodePasswordWithIterations(password, parts[2], parts[1])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(encoded)) == 1
}

func encodePasswordWithIterations(password, salt, iterations string) string {
	var count int
	_, _ = fmt.Sscanf(iterations, "%d", &count)
	if count <= 0 {
		count = passwordIterations
	}
	return fmt.Sprintf("sha256:%d:%s:%s", count, salt, derivePassword(password, salt, count))
}

func derivePassword(password, salt string, iterations int) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	for i := 1; i < iterations; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return hex.EncodeToString(sum[:])
}
