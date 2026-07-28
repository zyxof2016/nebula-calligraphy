package service

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type IdentityVerifier interface {
	CurrentUser(token string) (model.User, error)
}

type JWKSIdentityConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
	JWKS     []byte
}

type NebulaJWTIdentityConfig struct {
	Issuer   string
	Audience string
	Secret   string
}

type NebulaJWTIdentityVerifier struct {
	issuer   string
	audience string
	secret   []byte
	now      func() time.Time
}

type JWKSIdentityVerifier struct {
	issuer   string
	audience string
	jwksURL  string
	keys     map[string]*rsa.PublicKey
	mu       sync.RWMutex
	client   *http.Client
	now      func() time.Time
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewJWKSIdentityVerifier(cfg JWKSIdentityConfig) *JWKSIdentityVerifier {
	verifier := &JWKSIdentityVerifier{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		jwksURL:  cfg.JWKSURL,
		keys:     make(map[string]*rsa.PublicKey),
		client:   &http.Client{Timeout: 10 * time.Second},
		now:      time.Now,
	}
	if len(cfg.JWKS) > 0 {
		verifier.loadJWKS(cfg.JWKS)
	}
	return verifier
}

func NewNebulaJWTIdentityVerifier(cfg NebulaJWTIdentityConfig) *NebulaJWTIdentityVerifier {
	return &NebulaJWTIdentityVerifier{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		secret:   []byte(cfg.Secret),
		now:      time.Now,
	}
}

func (v *NebulaJWTIdentityVerifier) Check(context.Context) error {
	if strings.TrimSpace(v.issuer) == "" || strings.TrimSpace(v.audience) == "" || len(v.secret) < 32 {
		return errors.New("HS256 identity verifier is not securely configured")
	}
	return nil
}

func (v *JWKSIdentityVerifier) Check(ctx context.Context) error {
	if strings.TrimSpace(v.issuer) == "" || strings.TrimSpace(v.audience) == "" {
		return errors.New("JWKS identity issuer and audience are required")
	}
	if err := v.refreshJWKS(ctx); err != nil {
		return err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(v.keys) == 0 {
		return errors.New("JWKS endpoint returned no supported RS256 keys")
	}
	return nil
}

func (v *NebulaJWTIdentityVerifier) CurrentUser(token string) (model.User, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || len(v.secret) == 0 {
		return model.User{}, ErrUnauthorized
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "HS256" {
		return model.User{}, ErrUnauthorized
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return model.User{}, ErrUnauthorized
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	var claims struct {
		Issuer            string        `json:"iss"`
		Subject           string        `json:"sub"`
		UserID            string        `json:"uid"`
		PreferredUsername string        `json:"preferred_username"`
		Username          string        `json:"username"`
		ExpiresAt         float64       `json:"exp"`
		NotBefore         float64       `json:"nbf"`
		Audience          audienceClaim `json:"aud"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return model.User{}, ErrUnauthorized
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return model.User{}, ErrUnauthorized
	}
	if !claims.Audience.Contains(v.audience) || (claims.NotBefore > 0 && float64(v.now().Unix()) < claims.NotBefore) {
		return model.User{}, ErrUnauthorized
	}
	userID := claims.Subject
	if userID == "" {
		userID = claims.UserID
	}
	if userID == "" || claims.ExpiresAt <= float64(v.now().Unix()) {
		return model.User{}, ErrUnauthorized
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Username
	}
	if username == "" {
		username = userID
	}
	return model.User{UserID: userID, Username: username}, nil
}

func (v *JWKSIdentityVerifier) CurrentUser(token string) (model.User, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return model.User{}, ErrUnauthorized
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "RS256" {
		return model.User{}, ErrUnauthorized
	}
	key, err := v.findKey(header.Kid)
	if err != nil {
		return model.User{}, err
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	sum := sha256.Sum256([]byte(unsigned))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], signature); err != nil {
		return model.User{}, ErrUnauthorized
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.User{}, ErrUnauthorized
	}
	var claims struct {
		Issuer            string        `json:"iss"`
		Subject           string        `json:"sub"`
		PreferredUsername string        `json:"preferred_username"`
		ExpiresAt         float64       `json:"exp"`
		NotBefore         float64       `json:"nbf"`
		Audience          audienceClaim `json:"aud"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return model.User{}, ErrUnauthorized
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return model.User{}, ErrUnauthorized
	}
	if !claims.Audience.Contains(v.audience) || (claims.NotBefore > 0 && float64(v.now().Unix()) < claims.NotBefore) {
		return model.User{}, ErrUnauthorized
	}
	if claims.Subject == "" || claims.ExpiresAt <= float64(v.now().Unix()) {
		return model.User{}, ErrUnauthorized
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Subject
	}
	return model.User{UserID: claims.Subject, Username: username}, nil
}

type audienceClaim []string

func (a *audienceClaim) UnmarshalJSON(content []byte) error {
	var single string
	if err := json.Unmarshal(content, &single); err == nil {
		*a = audienceClaim{single}
		return nil
	}
	var multiple []string
	if err := json.Unmarshal(content, &multiple); err != nil {
		return err
	}
	*a = audienceClaim(multiple)
	return nil
}

func (a audienceClaim) Contains(expected string) bool {
	if strings.TrimSpace(expected) == "" {
		return false
	}
	for _, value := range a {
		if value == expected {
			return true
		}
	}
	return false
}

func (v *JWKSIdentityVerifier) findKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	v.mu.RUnlock()
	if ok {
		return key, nil
	}
	if strings.TrimSpace(v.jwksURL) == "" {
		return nil, ErrUnauthorized
	}
	if err := v.refreshJWKS(context.Background()); err != nil {
		return nil, fmt.Errorf("%w: refresh JWKS: %v", ErrIdentityUnavailable, err)
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, ErrUnauthorized
	}
	return key, nil
}

func (v *JWKSIdentityVerifier) refreshJWKS(ctx context.Context) error {
	if strings.TrimSpace(v.jwksURL) == "" {
		v.mu.RLock()
		hasKeys := len(v.keys) > 0
		v.mu.RUnlock()
		if hasKeys {
			return nil
		}
		return errors.New("JWKS URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("JWKS endpoint returned %s", resp.Status)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	v.loadJWKS(content)
	return nil
}

func (v *JWKSIdentityVerifier) loadJWKS(content []byte) {
	var doc jwksDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		return
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range doc.Keys {
		if key.Kty != "RSA" || key.Alg != "RS256" || key.Kid == "" {
			continue
		}
		if pub, ok := rsaPublicKeyFromJWK(key); ok {
			keys[key.Kid] = pub
		}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
}

func rsaPublicKeyFromJWK(key jwkKey) (*rsa.PublicKey, bool) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, false
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, false
	}
	e := big.NewInt(0).SetBytes(eBytes).Int64()
	if e <= 0 {
		return nil, false
	}
	return &rsa.PublicKey{N: big.NewInt(0).SetBytes(nBytes), E: int(e)}, true
}
