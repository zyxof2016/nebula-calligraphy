package service

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestNebulaJWTIdentityVerifierValidatesHS256Token(t *testing.T) {
	verifier := NewNebulaJWTIdentityVerifier(NebulaJWTIdentityConfig{
		Issuer:   "nebula",
		Audience: "nebula-calligraphy",
		Secret:   "test-secret-key-32bytes!!!!!!!!",
	})
	verifier.now = func() time.Time { return time.Unix(1800000000, 0) }
	token := signTestHMACJWT(t, "test-secret-key-32bytes!!!!!!!!", map[string]any{
		"iss":                "nebula",
		"sub":                "user-123",
		"uid":                "user-123",
		"preferred_username": "learner",
		"aud":                "nebula-calligraphy",
		"exp":                float64(1900000000),
	})

	user, err := verifier.CurrentUser(token)
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if user.UserID != "user-123" || user.Username != "learner" {
		t.Fatalf("user = %#v, want mapped Nebula JWT claims", user)
	}
}

func TestNebulaJWTIdentityVerifierRejectsWrongAudienceAndFutureToken(t *testing.T) {
	verifier := NewNebulaJWTIdentityVerifier(NebulaJWTIdentityConfig{
		Issuer:   "nebula",
		Audience: "nebula-calligraphy",
		Secret:   "test-secret-key-32bytes!!!!!!!!",
	})
	verifier.now = func() time.Time { return time.Unix(1800000000, 0) }

	for name, claims := range map[string]map[string]any{
		"wrong audience": {
			"iss": "nebula", "sub": "user-123", "aud": "another-service", "exp": float64(1900000000),
		},
		"future token": {
			"iss": "nebula", "sub": "user-123", "aud": "nebula-calligraphy", "nbf": float64(1800000060), "exp": float64(1900000000),
		},
	} {
		t.Run(name, func(t *testing.T) {
			token := signTestHMACJWT(t, "test-secret-key-32bytes!!!!!!!!", claims)
			if _, err := verifier.CurrentUser(token); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("CurrentUser() error = %v, want unauthorized", err)
			}
		})
	}
}

func TestJWKSIdentityVerifierValidatesRS256Token(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	jwks := testJWKS(t, &key.PublicKey, "kid-1")
	verifier := NewJWKSIdentityVerifier(JWKSIdentityConfig{
		Issuer:   "https://identity.example",
		Audience: "nebula-calligraphy",
		JWKS:     jwks,
	})
	verifier.now = func() time.Time { return time.Unix(1800000000, 0) }
	token := signTestJWT(t, key, map[string]any{
		"iss":                "https://identity.example",
		"sub":                "user-123",
		"preferred_username": "learner",
		"aud":                []string{"nebula-calligraphy", "nebula-platform"},
		"exp":                float64(1900000000),
	}, "kid-1")

	user, err := verifier.CurrentUser(token)
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if user.UserID != "user-123" || user.Username != "learner" {
		t.Fatalf("user = %#v, want mapped claims", user)
	}
}

func TestJWKSIdentityVerifierReplacesRevokedKeys(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(first) error = %v", err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(second) error = %v", err)
	}
	verifier := NewJWKSIdentityVerifier(JWKSIdentityConfig{
		Issuer:   "https://identity.example",
		Audience: "nebula-calligraphy",
		JWKS:     testJWKS(t, &first.PublicKey, "kid-1"),
	})

	verifier.loadJWKS(testJWKS(t, &second.PublicKey, "kid-2"))

	verifier.mu.RLock()
	defer verifier.mu.RUnlock()
	if _, ok := verifier.keys["kid-1"]; ok {
		t.Fatal("revoked kid-1 remains cached after JWKS replacement")
	}
	if _, ok := verifier.keys["kid-2"]; !ok {
		t.Fatal("replacement kid-2 is missing")
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
