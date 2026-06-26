package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type S3ArtifactStoreConfig struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type S3ArtifactStore struct {
	cfg    S3ArtifactStoreConfig
	client *http.Client
	now    func() time.Time
}

func NewS3ArtifactStore(cfg S3ArtifactStoreConfig) *S3ArtifactStore {
	return &S3ArtifactStore{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		now:    time.Now,
	}
}

func (s *S3ArtifactStore) Save(export model.ExportRecord, content string) (string, error) {
	if strings.TrimSpace(s.cfg.Endpoint) == "" || strings.TrimSpace(s.cfg.Bucket) == "" {
		return "", errors.New("s3 endpoint and bucket are required")
	}
	if strings.TrimSpace(s.cfg.Region) == "" {
		s.cfg.Region = "us-east-1"
	}
	if strings.TrimSpace(s.cfg.AccessKeyID) == "" || strings.TrimSpace(s.cfg.SecretAccessKey) == "" {
		return "", errors.New("s3 access key and secret are required")
	}
	key := path.Join(export.ArtworkID, export.ExportID+"."+export.Format)
	endpoint, err := url.Parse(strings.TrimRight(s.cfg.Endpoint, "/") + "/" + s.cfg.Bucket + "/" + key)
	if err != nil {
		return "", err
	}
	body := []byte(content)
	req, err := http.NewRequest(http.MethodPut, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", export.ContentType)
	if export.ContentType == "" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	s.sign(req, body)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("s3 put object failed: %s", resp.Status)
	}
	return key, nil
}

func (s *S3ArtifactStore) sign(req *http.Request, body []byte) {
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	payloadHash := sha256Hex(body)
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if s.cfg.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", s.cfg.SessionToken)
	}

	canonicalURI := req.URL.EscapedPath()
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.URL.Host, payloadHash, amzDate)
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := shortDate + "/" + s.cfg.Region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(s.cfg.SecretAccessKey, shortDate, s.cfg.Region), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", s.cfg.AccessKeyID, scope, signedHeaders, signature))
}

func signingKey(secret, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
