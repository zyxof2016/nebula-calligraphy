package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileAuditLoggerWritesJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	logger := NewFileAuditLogger(path)

	if err := logger.Record(AuditEvent{
		Action:     "auth.login",
		ActorID:    "user-1",
		ResourceID: "session-1",
		Outcome:    "success",
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var event AuditEvent
	if err := json.Unmarshal(content, &event); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; content=%q", err, content)
	}
	if event.Action != "auth.login" || event.ActorID != "user-1" || event.Outcome != "success" {
		t.Fatalf("event = %#v, want recorded audit event", event)
	}
	if event.CreatedAt == "" {
		t.Fatal("CreatedAt is empty")
	}
}

func TestHTTPAuditLoggerPostsAuditEvent(t *testing.T) {
	var gotMethod string
	var gotAuth string
	var gotEvent AuditEvent
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEvent); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer sink.Close()

	logger := NewHTTPAuditLogger(HTTPAuditLoggerConfig{
		Endpoint:    sink.URL + "/events",
		BearerToken: "audit-token",
	})

	if err := logger.Record(AuditEvent{
		Action:     "artwork.export",
		ActorID:    "user-1",
		ResourceID: "export-1",
		Outcome:    "success",
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotAuth != "Bearer audit-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
	if gotEvent.Action != "artwork.export" || gotEvent.ActorID != "user-1" || gotEvent.ResourceID != "export-1" {
		t.Fatalf("event = %#v, want posted audit event", gotEvent)
	}
	if gotEvent.CreatedAt == "" {
		t.Fatal("CreatedAt is empty")
	}
}

func TestHTTPAuditLoggerReturnsErrorForSinkFailure(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer sink.Close()

	logger := NewHTTPAuditLogger(HTTPAuditLoggerConfig{Endpoint: sink.URL})

	if err := logger.Record(AuditEvent{Action: "auth.login"}); err == nil {
		t.Fatal("Record() error = nil, want sink failure")
	}
}

func TestHTTPAuditLoggerChecksAuthenticatedHealthEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()

	logger := NewHTTPAuditLogger(HTTPAuditLoggerConfig{
		Endpoint:       sink.URL + "/events",
		HealthEndpoint: sink.URL + "/health/ready",
		BearerToken:    "audit-token",
	})
	if err := logger.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if gotPath != "/health/ready" || gotAuth != "Bearer audit-token" {
		t.Fatalf("health request path=%q auth=%q", gotPath, gotAuth)
	}
}

func TestHTTPAuditLoggerHealthCheckRejectsFailure(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer sink.Close()

	logger := NewHTTPAuditLogger(HTTPAuditLoggerConfig{HealthEndpoint: sink.URL})
	if err := logger.Check(context.Background()); err == nil {
		t.Fatal("Check() error = nil, want health failure")
	}
}
