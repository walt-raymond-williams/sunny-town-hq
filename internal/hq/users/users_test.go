package users

import (
	"context"
	"errors"
	"testing"

	hqauth "hq/internal/hq/auth"

	"github.com/jackc/pgx/v5"
)

func TestRequireReturnsContextUser(t *testing.T) {
	user := hqauth.User{
		ID:              12,
		KeycloakSubject: "subject",
		DisplayName:     "Student",
		Roles:           []string{"student"},
	}

	got, err := Require(hqauth.WithUser(context.Background(), user))
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
	if got.ID != user.ID || got.KeycloakSubject != user.KeycloakSubject {
		t.Fatalf("Require() = %#v, want %#v", got, user)
	}
}

func TestRequireRejectsMissingUser(t *testing.T) {
	_, err := Require(context.Background())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Require() error = %v, want %v", err, pgx.ErrNoRows)
	}
}

func TestSyncAuthenticatedRejectsMissingSubjectBeforeDB(t *testing.T) {
	store := NewStore(nil)

	_, err := store.SyncAuthenticated(context.Background(), hqauth.User{DisplayName: "No Subject"})
	if !errors.Is(err, hqauth.ErrInvalidToken) {
		t.Fatalf("SyncAuthenticated() error = %v, want %v", err, hqauth.ErrInvalidToken)
	}
}
