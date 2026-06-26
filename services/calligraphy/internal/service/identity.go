package service

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type IdentityVerifier interface {
	CurrentUser(token string) (model.User, bool)
}

type JWKSIdentityConfig struct {
	Issuer  string
	JWKSURL string
	JWKS    []byte
}

type NebulaJWTIdentityConfig struct {
	Issuer string
	Secret string
}

type NebulaJWTIdentityVerifier struct {
	issuer string
	secret []byte
	now    func() time.Time
}

type JWKSIdentityVerifier struct {
	issuer  string
	jwksURL string
	keys    map[string]*rsa.PublicKey
	mu      sync.RWMutex
	client  *http.Client
	now     func() time.Time
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
		issuer:  cfg.Issuer,
		jwksURL: cfg.JWKSURL,
		keys:    make(map[string]*rsa.PublicKey),
		client:  &http.Client{Timeout: 10 * time.Second},
		now:     time.Now,
	}
	if len(cfg.JWKS) > 0 {
		verifier.loadJWKS(cfg.JWKS)
	}
	return verifier
}

func NewNebulaJWTIdentityVerifier(cfg NebulaJWTIdentityConfig) *NebulaJWTIdentityVerifier {
	return &NebulaJWTIdentityVerifier{
		issuer: cfg.Issuer,
		secret: []byte(cfg.Secret),
		now:    time.Now,
	}
}

func (v *NebulaJWTIdentityVerifier) CurrentUser(token string) (model.User, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || len(v.secret) == 0 {
		return model.User{}, false
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.User{}, false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "HS256" {
		return model.User{}, false
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.User{}, false
	}
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return model.User{}, false
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.User{}, false
	}
	var claims struct {
		Issuer            string  `json:"iss"`
		Subject           string  `json:"sub"`
		UserID            string  `json:"uid"`
		PreferredUsername string  `json:"preferred_username"`
		Username          string  `json:"username"`
		ExpiresAt         float64 `json:"exp"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return model.User{}, false
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return model.User{}, false
	}
	userID := claims.Subject
	if userID == "" {
		userID = claims.UserID
	}
	if userID == "" || claims.ExpiresAt <= float64(v.now().Unix()) {
		return model.User{}, false
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Username
	}
	if username == "" {
		username = userID
	}
	return model.User{UserID: userID, Username: username}, true
}

func (v *JWKSIdentityVerifier) CurrentUser(token string) (model.User, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return model.User{}, false
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.User{}, false
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "RS256" {
		return model.User{}, false
	}
	key, ok := v.findKey(header.Kid)
	if !ok {
		return model.User{}, false
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.User{}, false
	}
	sum := sha256.Sum256([]byte(unsigned))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], signature); err != nil {
		return model.User{}, false
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.User{}, false
	}
	var claims struct {
		Issuer            string  `json:"iss"`
		Subject           string  `json:"sub"`
		PreferredUsername string  `json:"preferred_username"`
		ExpiresAt         float64 `json:"exp"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return model.User{}, false
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return model.User{}, false
	}
	if claims.Subject == "" || claims.ExpiresAt <= float64(v.now().Unix()) {
		return model.User{}, false
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Subject
	}
	return model.User{UserID: claims.Subject, Username: username}, true
}

func (v *JWKSIdentityVerifier) findKey(kid string) (*rsa.PublicKey, bool) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	v.mu.RUnlock()
	if ok {
		return key, true
	}
	if strings.TrimSpace(v.jwksURL) == "" {
		return nil, false
	}
	resp, err := v.client.Get(v.jwksURL)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, false
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return nil, false
	}
	v.loadJWKS(content)
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	return key, ok
}

func (v *JWKSIdentityVerifier) loadJWKS(content []byte) {
	var doc jwksDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, key := range doc.Keys {
		if key.Kty != "RSA" || key.Alg != "RS256" || key.Kid == "" {
			continue
		}
		if pub, ok := rsaPublicKeyFromJWK(key); ok {
			v.keys[key.Kid] = pub
		}
	}
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
