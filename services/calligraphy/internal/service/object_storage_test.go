package service

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

func TestS3ArtifactStoreSavesExportWithSigV4(t *testing.T) {
	var method, path, authHeader, body string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		method = r.Method
		path = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		content, _ := io.ReadAll(r.Body)
		body = string(content)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	store := NewS3ArtifactStore(S3ArtifactStoreConfig{
		Endpoint:        "https://s3.example.test",
		Bucket:          "calligraphy-prod",
		Region:          "us-east-1",
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "secret",
	})
	store.now = func() time.Time { return time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC) }
	store.client = &http.Client{Transport: transport}

	key, err := store.Save(model.ExportRecord{
		ArtworkID: "artwork-000001",
		ExportID:  "export-000001",
		Format:    "svg",
	}, "<svg></svg>")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if key != "artwork-000001/export-000001.svg" {
		t.Fatalf("key = %q, want artwork-000001/export-000001.svg", key)
	}
	if method != http.MethodPut {
		t.Fatalf("method = %q, want PUT", method)
	}
	if path != "/calligraphy-prod/artwork-000001/export-000001.svg" {
		t.Fatalf("path = %q, want bucket object path", path)
	}
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("Authorization = %q, want AWS4-HMAC-SHA256", authHeader)
	}
	if body != "<svg></svg>" {
		t.Fatalf("body = %q, want svg content", body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
