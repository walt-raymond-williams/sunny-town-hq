package serviceauth

import (
	"net/http"
	"strings"
)

const (
	HQServiceNameHeader = "X-HQ-Service-Name"
	HQServiceSecret     = "X-HQ-Service-Secret"
	AIServiceSecret     = "X-AI-Service-Secret"
)

func Authorized(header http.Header, secretHeader string, expectedSecret string) bool {
	expectedSecret = strings.TrimSpace(expectedSecret)
	if expectedSecret == "" {
		return false
	}
	return strings.TrimSpace(header.Get(secretHeader)) == expectedSecret
}
