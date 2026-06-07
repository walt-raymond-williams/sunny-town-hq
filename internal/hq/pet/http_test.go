package pet

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPHandlerRequiresStudentRole(t *testing.T) {
	handler := NewHTTPHandler(HTTPHandlerConfig{
		Store: fakeHTTPStore{},
		RequireRole: func(w http.ResponseWriter, _ *http.Request, role string) (RoleUser, bool) {
			if role != "student" {
				t.Fatalf("role = %q, want student", role)
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "student role required"})
			return RoleUser{}, false
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/student/profile", nil)
	response := httptest.NewRecorder()

	handler.HandleStudentProfile(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestHTTPHandlerRejectsWrongMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		handle func(HTTPHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "profile only allows get",
			method: http.MethodPost,
			path:   "/api/student/profile",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentProfile(w, r)
			},
		},
		{
			name:   "feed only allows post",
			method: http.MethodGet,
			path:   "/api/student/pet/feed",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleFeedStudentPet(w, r)
			},
		},
	}

	handler := handlerWithStudentRole(fakeHTTPStore{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()

			test.handle(handler, response, request)

			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHTTPHandlerReturnsProfile(t *testing.T) {
	handler := handlerWithStudentRole(fakeHTTPStore{
		profile: Profile{ID: 123, DisplayName: "Student", Cookies: 2, StarBalance: 7},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/student/profile", nil)
	response := httptest.NewRecorder()

	handler.HandleStudentProfile(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, field := range []string{`"id":123`, `"display_name":"Student"`, `"star_balance":7`} {
		if !strings.Contains(body, field) {
			t.Fatalf("body = %s, want field %s", body, field)
		}
	}
}

func TestHTTPHandlerMapsNoCookies(t *testing.T) {
	noCookies := errors.New("no cookies available")
	handler := handlerWithStudentRole(fakeHTTPStore{feedErr: noCookies})
	handler.noCookiesError = noCookies

	request := httptest.NewRequest(http.MethodPost, "/api/student/pet/feed", nil)
	response := httptest.NewRecorder()

	handler.HandleFeedStudentPet(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "no cookies available") {
		t.Fatalf("body = %q, want no cookies error", response.Body.String())
	}
}

func handlerWithStudentRole(store HTTPStore) HTTPHandler {
	return NewHTTPHandler(HTTPHandlerConfig{
		Store: store,
		RequireRole: func(_ http.ResponseWriter, _ *http.Request, role string) (RoleUser, bool) {
			if role != "student" {
				return RoleUser{}, false
			}
			return RoleUser{ID: 123}, true
		},
	})
}

type fakeHTTPStore struct {
	profile Profile
	loadErr error
	feedErr error
}

func (store fakeHTTPStore) LoadProfile(context.Context, int64) (Profile, error) {
	return store.profile, store.loadErr
}

func (store fakeHTTPStore) Feed(context.Context, int64) (Profile, error) {
	return store.profile, store.feedErr
}
