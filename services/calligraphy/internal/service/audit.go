package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type AuditLogger interface {
	Record(event AuditEvent) error
}

type AuditEvent struct {
	CreatedAt  string            `json:"created_at"`
	Action     string            `json:"action"`
	ActorID    string            `json:"actor_id,omitempty"`
	ResourceID string            `json:"resource_id,omitempty"`
	Outcome    string            `json:"outcome"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type NoopAuditLogger struct{}

func (NoopAuditLogger) Record(AuditEvent) error {
	return nil
}

type FileAuditLogger struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

type HTTPAuditLoggerConfig struct {
	Endpoint       string
	HealthEndpoint string
	BearerToken    string
}

type HTTPAuditLogger struct {
	endpoint       string
	healthEndpoint string
	bearerToken    string
	client         *http.Client
	now            func() time.Time
}

func NewFileAuditLogger(path string) *FileAuditLogger {
	return &FileAuditLogger{path: path, now: time.Now}
}

func NewHTTPAuditLogger(cfg HTTPAuditLoggerConfig) *HTTPAuditLogger {
	return &HTTPAuditLogger{
		endpoint:       strings.TrimSpace(cfg.Endpoint),
		healthEndpoint: strings.TrimSpace(cfg.HealthEndpoint),
		bearerToken:    strings.TrimSpace(cfg.BearerToken),
		client:         &http.Client{Timeout: 5 * time.Second},
		now:            time.Now,
	}
}

func (l *FileAuditLogger) Record(event AuditEvent) error {
	if strings.TrimSpace(l.path) == "" {
		return errors.New("audit log path is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.CreatedAt == "" {
		event.CreatedAt = l.now().UTC().Format(time.RFC3339)
	}
	if event.Outcome == "" {
		event.Outcome = "success"
	}
	content, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(content, '\n')); err != nil {
		return err
	}
	return nil
}

func (l *HTTPAuditLogger) Record(event AuditEvent) error {
	if strings.TrimSpace(l.endpoint) == "" {
		return errors.New("audit sink endpoint is required")
	}
	if event.CreatedAt == "" {
		event.CreatedAt = l.now().UTC().Format(time.RFC3339)
	}
	if event.Outcome == "" {
		event.Outcome = "success"
	}
	content, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, l.endpoint, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.bearerToken)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit sink returned status %d", resp.StatusCode)
	}
	return nil
}

func (l *HTTPAuditLogger) Check(ctx context.Context) error {
	if strings.TrimSpace(l.healthEndpoint) == "" {
		return errors.New("audit sink health endpoint is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.healthEndpoint, nil)
	if err != nil {
		return err
	}
	if l.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.bearerToken)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("check audit sink health: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit sink health returned status %d", resp.StatusCode)
	}
	return nil
}
