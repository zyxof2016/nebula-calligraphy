package service

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestNebulaJWTIdentityVerifierValidatesHS256Token(t *testing.T) {
	verifier := NewNebulaJWTIdentityVerifier(NebulaJWTIdentityConfig{
		Issuer: "nebula",
		Secret: "test-secret-key-32bytes!!!!!!!!",
	})
	verifier.now = func() time.Time { return time.Unix(1800000000, 0) }
	token := signTestHMACJWT(t, "test-secret-key-32bytes!!!!!!!!", map[string]any{
		"iss":                "nebula",
		"sub":                "user-123",
		"uid":                "user-123",
		"preferred_username": "learner",
		"exp":                float64(1900000000),
	})

	user, ok := verifier.CurrentUser(token)
	if !ok {
		t.Fatal("CurrentUser() ok = false, want true")
	}
	if user.UserID != "user-123" || user.Username != "learner" {
		t.Fatalf("user = %#v, want mapped Nebula JWT claims", user)
	}
}

func TestJWKSIdentityVerifierValidatesRS256Token(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	jwks := testJWKS(t, &key.PublicKey, "kid-1")
	verifier := NewJWKSIdentityVerifier(JWKSIdentityConfig{
		Issuer: "https://identity.example",
		JWKS:   jwks,
	})
	verifier.now = func() time.Time { return time.Unix(1800000000, 0) }
	token := signTestJWT(t, key, map[string]any{
		"iss":                "https://identity.example",
		"sub":                "user-123",
		"preferred_username": "learner",
		"exp":                float64(1900000000),
	}, "kid-1")

	user, ok := verifier.CurrentUser(token)
	if !ok {
		t.Fatal("CurrentUser() ok = false, want true")
	}
	if user.UserID != "user-123" || user.Username != "learner" {
		t.Fatalf("user = %#v, want mapped claims", user)
	}
}

func signTestHMACJWT(t *testing.T, secret string, claims map[string]any) string {
	t.Helper()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerBytes, _ := json.Marshal(header)
	claimBytes, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any, kid string) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	headerBytes, _ := json.Marshal(header)
	claimBytes, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimBytes)
	sum := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func testJWKS(t *testing.T, key *rsa.PublicKey, kid string) []byte {
	t.Helper()
	body := map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": kid,
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}},
	}
	content, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal(jwks) error = %v", err)
	}
	if !strings.Contains(string(content), kid) {
		t.Fatalf("jwks does not contain kid")
	}
	return content
}
