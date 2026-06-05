package sunnytownauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid sunny town token")
	ErrExpiredToken = errors.New("expired sunny town token")
)

type Claims struct {
	AppUserID       int64    `json:"app_user_id"`
	KeycloakSubject string   `json:"sub"`
	DisplayName     string   `json:"display_name"`
	Roles           []string `json:"roles"`
	RoomID          string   `json:"room_id"`
	MapID           string   `json:"map_id"`
	AvatarID        string   `json:"avatar_id"`
	ExpiresAt       int64    `json:"exp"`
}

func Sign(claims Claims, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("sunny town join secret is required")
	}
	if claims.ExpiresAt == 0 {
		return "", fmt.Errorf("sunny town token expiration is required")
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(encodedPayload, secret)
	return encodedPayload + "." + signature, nil
}

func Verify(token string, secret string, now time.Time) (Claims, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return Claims{}, ErrInvalidToken
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Claims{}, ErrInvalidToken
	}

	expected := sign(parts[0], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.ExpiresAt <= now.Unix() {
		return Claims{}, ErrExpiredToken
	}
	if claims.AppUserID < 1 || claims.KeycloakSubject == "" || claims.RoomID == "" || claims.MapID == "" {
		return Claims{}, ErrInvalidToken
	}
	if !HasRole(claims.Roles, "student") {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

func HasRole(roles []string, role string) bool {
	for _, value := range roles {
		if value == role {
			return true
		}
	}
	return false
}

func PlayerID(appUserID int64) string {
	return strconv.FormatInt(appUserID, 10)
}

func sign(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
