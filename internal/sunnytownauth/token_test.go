package sunnytownauth

import (
	"errors"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := Sign(testClaims(now.Add(time.Minute)), "secret")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	claims, err := Verify(token, "secret", now)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.AppUserID != 42 || claims.CharacterID != 420 || claims.RoomID != "sunny-town-main" {
		t.Fatalf("Verify() claims = %#v", claims)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := Sign(testClaims(now.Add(-time.Second)), "secret")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	_, err = Verify(token, "secret", now)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := Sign(testClaims(now.Add(time.Minute)), "secret")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	_, err = Verify(token, "wrong", now)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestVerifyRequiresStudentRole(t *testing.T) {
	now := time.Unix(1000, 0)
	claims := testClaims(now.Add(time.Minute))
	claims.Roles = []string{"teacher"}
	token, err := Sign(claims, "secret")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	_, err = Verify(token, "secret", now)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
	}
}

func testClaims(expiresAt time.Time) Claims {
	return Claims{
		AppUserID:       42,
		CharacterID:     420,
		KeycloakSubject: "subject",
		DisplayName:     "Student",
		Roles:           []string{"student"},
		RoomID:          "sunny-town-main",
		MapID:           "sunny-town-v1",
		AvatarID:        "pet-default",
		ExpiresAt:       expiresAt.Unix(),
	}
}
