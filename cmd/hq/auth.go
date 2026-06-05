package main

import (
	"context"
	"crypto"
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
)

type contextKey string

const authUserContextKey contextKey = "authUser"

var (
	errMissingBearer = errors.New("missing bearer token")
	errInvalidToken  = errors.New("invalid token")
)

type authVerifier struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client
	mu       sync.Mutex
	keys     map[string]*rsa.PublicKey
	fetched  time.Time
}

type authUser struct {
	ID              int64    `json:"id"`
	KeycloakSubject string   `json:"keycloak_subject"`
	DisplayName     string   `json:"display_name"`
	Email           string   `json:"email,omitempty"`
	Roles           []string `json:"roles"`
}

type tokenClaims struct {
	Subject           string               `json:"sub"`
	Issuer            string               `json:"iss"`
	Audience          audienceClaim        `json:"aud"`
	AuthorizedParty   string               `json:"azp"`
	ExpiresAt         int64                `json:"exp"`
	NotBefore         int64                `json:"nbf"`
	IssuedAt          int64                `json:"iat"`
	Email             string               `json:"email"`
	Name              string               `json:"name"`
	PreferredUsername string               `json:"preferred_username"`
	RealmAccess       roleClaim            `json:"realm_access"`
	ResourceAccess    map[string]roleClaim `json:"resource_access"`
}

type roleClaim struct {
	Roles []string `json:"roles"`
}

type audienceClaim []string

func (claim *audienceClaim) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*claim = []string{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*claim = many
	return nil
}

func newAuthVerifier(issuer string, audience string, jwksURL string) *authVerifier {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	jwksURL = strings.TrimSpace(jwksURL)
	if jwksURL == "" {
		jwksURL = issuer + "/protocol/openid-connect/certs"
	}

	return &authVerifier{
		issuer:   issuer,
		audience: strings.TrimSpace(audience),
		jwksURL:  jwksURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		keys: map[string]*rsa.PublicKey{},
	}
}

func (verifier *authVerifier) authenticateRequest(ctx context.Context, r *http.Request) (authUser, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return authUser{}, errMissingBearer
	}

	claims, err := verifier.verifyToken(ctx, strings.TrimSpace(header[len("Bearer "):]))
	if err != nil {
		return authUser{}, err
	}

	roles := claims.rolesForAudience(verifier.audience)
	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(claims.PreferredUsername)
	}
	if displayName == "" {
		displayName = strings.TrimSpace(claims.Email)
	}
	if displayName == "" {
		displayName = claims.Subject
	}

	return authUser{
		KeycloakSubject: claims.Subject,
		DisplayName:     displayName,
		Email:           strings.TrimSpace(claims.Email),
		Roles:           roles,
	}, nil
}

func (verifier *authVerifier) verifyToken(ctx context.Context, token string) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, errInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return tokenClaims{}, err
	}
	if header.Algorithm != "RS256" || header.KeyID == "" {
		return tokenClaims{}, errInvalidToken
	}

	key, err := verifier.publicKey(ctx, header.KeyID)
	if err != nil {
		return tokenClaims{}, err
	}

	signed := []byte(parts[0] + "." + parts[1])
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return tokenClaims{}, errInvalidToken
	}
	hash := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		return tokenClaims{}, errInvalidToken
	}

	var claims tokenClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return tokenClaims{}, err
	}
	if claims.Subject == "" || claims.Issuer != verifier.issuer {
		return tokenClaims{}, errInvalidToken
	}

	now := time.Now().Unix()
	if claims.ExpiresAt <= now || (claims.NotBefore > 0 && claims.NotBefore > now) {
		return tokenClaims{}, errInvalidToken
	}
	if verifier.audience != "" && !claims.hasAudience(verifier.audience) && claims.AuthorizedParty != verifier.audience {
		return tokenClaims{}, errInvalidToken
	}

	return claims, nil
}

func decodeJWTPart(part string, target any) error {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return errInvalidToken
	}
	if err := json.Unmarshal(data, target); err != nil {
		return errInvalidToken
	}
	return nil
}

func (verifier *authVerifier) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	verifier.mu.Lock()
	key := verifier.keys[kid]
	stale := time.Since(verifier.fetched) > 10*time.Minute
	verifier.mu.Unlock()

	if key != nil && !stale {
		return key, nil
	}

	if err := verifier.refreshKeys(ctx); err != nil {
		return nil, err
	}

	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	key = verifier.keys[kid]
	if key == nil {
		return nil, errInvalidToken
	}
	return key, nil
}

func (verifier *authVerifier) refreshKeys(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, verifier.jwksURL, nil)
	if err != nil {
		return err
	}

	response, err := verifier.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch keycloak jwks: status %d", response.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			KeyID     string `json:"kid"`
			KeyType   string `json:"kty"`
			Algorithm string `json:"alg"`
			Use       string `json:"use"`
			N         string `json:"n"`
			E         string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(response.Body).Decode(&jwks); err != nil {
		return err
	}

	keys := map[string]*rsa.PublicKey{}
	for _, jwk := range jwks.Keys {
		if jwk.KeyID == "" || jwk.KeyType != "RSA" || jwk.N == "" || jwk.E == "" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			continue
		}

		exponent := 0
		for _, b := range eBytes {
			exponent = exponent<<8 + int(b)
		}
		if exponent == 0 {
			continue
		}

		keys[jwk.KeyID] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: exponent,
		}
	}

	verifier.mu.Lock()
	verifier.keys = keys
	verifier.fetched = time.Now()
	verifier.mu.Unlock()
	return nil
}

func (claims tokenClaims) hasAudience(audience string) bool {
	for _, value := range claims.Audience {
		if value == audience {
			return true
		}
	}
	return false
}

func (claims tokenClaims) rolesForAudience(audience string) []string {
	seen := map[string]bool{}
	for _, role := range claims.RealmAccess.Roles {
		if role == "teacher" || role == "student" {
			seen[role] = true
		}
	}
	if resource, ok := claims.ResourceAccess[audience]; ok {
		for _, role := range resource.Roles {
			if role == "teacher" || role == "student" {
				seen[role] = true
			}
		}
	}

	roles := []string{}
	for _, role := range []string{"teacher", "student"} {
		if seen[role] {
			roles = append(roles, role)
		}
	}
	return roles
}

func userFromContext(ctx context.Context) (authUser, bool) {
	user, ok := ctx.Value(authUserContextKey).(authUser)
	return user, ok
}

func hasRole(user authUser, role string) bool {
	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}
