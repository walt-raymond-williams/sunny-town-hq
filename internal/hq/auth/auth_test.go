package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithUserAndUserFromContext(t *testing.T) {
	user := User{
		ID:              42,
		KeycloakSubject: "kc-subject",
		DisplayName:     "Ada",
		Roles:           []string{"student"},
	}

	got, ok := UserFromContext(WithUser(context.Background(), user))
	if !ok {
		t.Fatal("expected user in context")
	}
	if got.ID != user.ID || got.KeycloakSubject != user.KeycloakSubject || got.DisplayName != user.DisplayName {
		t.Fatalf("unexpected user: %#v", got)
	}
}

func TestHasRole(t *testing.T) {
	user := User{Roles: []string{"student"}}
	if !HasRole(user, "student") {
		t.Fatal("expected student role")
	}
	if HasRole(user, "teacher") {
		t.Fatal("did not expect teacher role")
	}
}

func TestRequireRole(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	user := User{ID: 7, Roles: []string{"teacher"}}
	request = request.WithContext(WithUser(request.Context(), user))
	response := httptest.NewRecorder()

	got, ok := RequireRole(response, request, "teacher")
	if !ok {
		t.Fatal("expected role check to pass")
	}
	if got.ID != user.ID {
		t.Fatalf("RequireRole() ID = %d, want %d", got.ID, user.ID)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected response status: %d", response.Code)
	}
}

func TestRequireRoleRejectsMissingLogin(t *testing.T) {
	response := httptest.NewRecorder()
	_, ok := RequireRole(response, httptest.NewRequest(http.MethodGet, "/", nil), "teacher")
	if ok {
		t.Fatal("expected role check to fail")
	}
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoleRejectsWrongRole(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(WithUser(request.Context(), User{ID: 7, Roles: []string{"student"}}))
	response := httptest.NewRecorder()

	_, ok := RequireRole(response, request, "teacher")
	if ok {
		t.Fatal("expected role check to fail")
	}
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestRequireStudentID(t *testing.T) {
	user := User{ID: 7, Roles: []string{"student"}}

	got, err := RequireStudentID(WithUser(context.Background(), user))
	if err != nil {
		t.Fatalf("RequireStudentID() error = %v", err)
	}
	if got != user.ID {
		t.Fatalf("RequireStudentID() = %d, want %d", got, user.ID)
	}
}

func TestRequireStudentIDRejectsMissingStudentRole(t *testing.T) {
	_, err := RequireStudentID(WithUser(context.Background(), User{ID: 7, Roles: []string{"teacher"}}))
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("RequireStudentID() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestRolesForAudienceKeepsKnownRolesInStableOrder(t *testing.T) {
	claims := tokenClaims{
		RealmAccess: roleClaim{Roles: []string{"ignored", "student"}},
		ResourceAccess: map[string]roleClaim{
			"hq": {Roles: []string{"teacher", "also-ignored"}},
		},
	}

	got := claims.rolesForAudience("hq")
	want := []string{"teacher", "student"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("rolesForAudience() = %v, want %v", got, want)
	}
}

func TestAuthenticateRequestRejectsMissingBearer(t *testing.T) {
	verifier := NewVerifier("http://issuer", "hq", "")

	_, err := verifier.AuthenticateRequest(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !errors.Is(err, ErrMissingBearer) {
		t.Fatalf("AuthenticateRequest() error = %v, want %v", err, ErrMissingBearer)
	}
}

func TestAuthenticateRequestWithSignedToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	const kid = "test-key"
	issuer := "http://issuer.example"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJWKS(t, w, kid, &privateKey.PublicKey)
	}))
	defer jwks.Close()

	verifier := NewVerifier(issuer, "hq", jwks.URL)
	token := signedTestJWT(t, privateKey, kid, map[string]any{
		"sub":                "subject-1",
		"iss":                issuer,
		"aud":                []string{"hq"},
		"exp":                time.Now().Add(time.Hour).Unix(),
		"nbf":                time.Now().Add(-time.Minute).Unix(),
		"email":              "ada@example.test",
		"preferred_username": "ada",
		"realm_access": map[string]any{
			"roles": []string{"student", "ignored"},
		},
		"resource_access": map[string]any{
			"hq": map[string]any{
				"roles": []string{"teacher"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	user, err := verifier.AuthenticateRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("AuthenticateRequest() error = %v", err)
	}
	if user.KeycloakSubject != "subject-1" || user.DisplayName != "ada" || user.Email != "ada@example.test" {
		t.Fatalf("unexpected user: %#v", user)
	}
	if strings.Join(user.Roles, ",") != "teacher,student" {
		t.Fatalf("unexpected roles: %v", user.Roles)
	}
}

func writeJWKS(t *testing.T, w http.ResponseWriter, kid string, publicKey *rsa.PublicKey) {
	t.Helper()

	exponent := big.NewInt(int64(publicKey.E)).Bytes()
	response := map[string]any{
		"keys": []map[string]string{
			{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("write jwks: %v", err)
	}
}

func signedTestJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()

	header := map[string]string{
		"alg": "RS256",
		"kid": kid,
		"typ": "JWT",
	}
	headerPart := encodeJWTPart(t, header)
	claimsPart := encodeJWTPart(t, claims)
	signed := headerPart + "." + claimsPart
	hash := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func encodeJWTPart(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal jwt part: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
