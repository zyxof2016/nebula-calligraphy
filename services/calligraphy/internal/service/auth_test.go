package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

func TestAuthServiceRegisterLoginAndCurrentUser(t *testing.T) {
	svc := NewAuthService(NewInMemoryAuthStore())
	svc.now = func() time.Time { return time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC) }
	svc.tokenSource = func() (string, error) { return "session-token-1", nil }
	svc.saltSource = func() (string, error) { return "salt-1", nil }

	session, err := svc.Register(model.AuthRequest{Username: " learner ", Password: "secret123"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if session.Token != "session-token-1" {
		t.Fatalf("Token = %q, want session-token-1", session.Token)
	}
	if session.ExpiresAt != "2026-06-26T09:00:00Z" {
		t.Fatalf("ExpiresAt = %q, want 24 hour expiry", session.ExpiresAt)
	}
	if session.User.UserID == "" || session.User.Username != "learner" {
		t.Fatalf("User = %#v, want normalized learner with id", session.User)
	}

	loaded, err := svc.CurrentUser(session.Token)
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if loaded.UserID != session.User.UserID {
		t.Fatalf("CurrentUser UserID = %q, want %q", loaded.UserID, session.User.UserID)
	}

	svc.tokenSource = func() (string, error) { return "session-token-2", nil }
	login, err := svc.Login(model.AuthRequest{Username: "learner", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if login.Token != "session-token-2" {
		t.Fatalf("login token = %q, want session-token-2", login.Token)
	}

	loggedOut, err := svc.Logout(login.Token)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !loggedOut {
		t.Fatal("Logout() = false, want true")
	}
	if _, err := svc.CurrentUser(login.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CurrentUser(logged out token) error = %v, want unauthorized", err)
	}
}

func TestAuthServiceExpiresSessionsAndUsesArgon2ID(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	store := NewInMemoryAuthStore()
	svc := NewAuthService(store)
	svc.now = func() time.Time { return now }
	svc.SetSessionTTL(time.Hour)
	svc.tokenSource = func() (string, error) { return "expiring-token", nil }
	svc.saltSource = func() (string, error) { return "test-salt", nil }

	session, err := svc.Register(model.AuthRequest{Username: "learner", Password: "secret123"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	stored, ok, err := store.FindUserByUsername("learner")
	if err != nil || !ok || !strings.HasPrefix(stored.PasswordHash, "argon2id:") {
		t.Fatalf("PasswordHash = %q, want argon2id encoding", stored.PasswordHash)
	}
	if _, err := svc.CurrentUser(session.Token); err != nil {
		t.Fatalf("CurrentUser() before expiry error = %v", err)
	}

	now = now.Add(time.Hour)
	if _, err := svc.CurrentUser(session.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CurrentUser() at expiry error = %v, want unauthorized", err)
	}
	if _, ok, err := store.FindSession(session.Token); err != nil || ok {
		t.Fatal("expired session remains in store")
	}
}

func TestAuthServiceRejectsDuplicateAndWrongPassword(t *testing.T) {
	svc := NewAuthService(NewInMemoryAuthStore())
	svc.tokenSource = func() (string, error) { return "session-token", nil }
	svc.saltSource = func() (string, error) { return "salt", nil }

	if _, err := svc.Register(model.AuthRequest{Username: "learner", Password: "secret123"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := svc.Register(model.AuthRequest{Username: "learner", Password: "secret123"}); err == nil {
		t.Fatal("Register duplicate error = nil, want error")
	}
	if _, err := svc.Login(model.AuthRequest{Username: "learner", Password: "badpass"}); err == nil {
		t.Fatal("Login wrong password error = nil, want error")
	}
}

func TestAuthServiceLocksLoginAfterRepeatedFailures(t *testing.T) {
	now := time.Date(2026, 6, 26, 8, 0, 0, 0, time.UTC)
	svc := NewAuthService(NewInMemoryAuthStore())
	svc.now = func() time.Time { return now }
	svc.tokenSource = func() (string, error) { return "session-token", nil }
	svc.saltSource = func() (string, error) { return "salt", nil }

	if _, err := svc.Register(model.AuthRequest{Username: "learner", Password: "secret123"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := svc.Login(model.AuthRequest{Username: "learner", Password: "wrongpass"}); err == nil {
			t.Fatalf("Login wrong password attempt %d error = nil, want error", i+1)
		}
	}
	if _, err := svc.Login(model.AuthRequest{Username: "learner", Password: "secret123"}); err == nil {
		t.Fatal("Login while locked error = nil, want error")
	}

	now = now.Add(16 * time.Minute)
	if _, err := svc.Login(model.AuthRequest{Username: "learner", Password: "secret123"}); err != nil {
		t.Fatalf("Login after lock window error = %v", err)
	}
}

func TestFileAuthStorePersistsUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	store, err := NewFileAuthStore(path)
	if err != nil {
		t.Fatalf("NewFileAuthStore() error = %v", err)
	}
	svc := NewAuthService(store)
	svc.tokenSource = func() (string, error) { return "session-token-1", nil }
	svc.saltSource = func() (string, error) { return "salt-1", nil }
	registered, err := svc.Register(model.AuthRequest{Username: "learner", Password: "secret123"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	reloadedStore, err := NewFileAuthStore(path)
	if err != nil {
		t.Fatalf("NewFileAuthStore(reload) error = %v", err)
	}
	reloaded := NewAuthService(reloadedStore)
	reloaded.tokenSource = func() (string, error) { return "session-token-2", nil }
	login, err := reloaded.Login(model.AuthRequest{Username: "learner", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login(reloaded) error = %v", err)
	}
	if login.User.UserID != registered.User.UserID {
		t.Fatalf("reloaded UserID = %q, want %q", login.User.UserID, registered.User.UserID)
	}
}
