package assignments

import (
	"context"
	"net/http"
	"strings"

	hqauth "hq/internal/hq/auth"
)

type RoleUser = hqauth.RoleUser
type RequireRoleFunc = hqauth.RequireRoleFunc
type GradeAttemptFunc func(context.Context, GradeAttemptCommand) error

type HTTPHandler struct {
	store                      Store
	requireRole                RequireRoleFunc
	gradeAttempt               GradeAttemptFunc
	triggerAIGradingForAttempt func(int64)
}

type HTTPHandlerConfig struct {
	Store                      Store
	RequireRole                RequireRoleFunc
	GradeAttempt               GradeAttemptFunc
	TriggerAIGradingForAttempt func(int64)
}

func NewHTTPHandler(config HTTPHandlerConfig) HTTPHandler {
	return HTTPHandler{
		store:                      config.Store,
		requireRole:                config.RequireRole,
		gradeAttempt:               config.GradeAttempt,
		triggerAIGradingForAttempt: config.TriggerAIGradingForAttempt,
	}
}

func (handler HTTPHandler) HandleAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := handler.requireRole(w, r, "teacher"); !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		handler.listAssignments(w, r)
	case http.MethodPost:
		handler.createAssignment(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler HTTPHandler) HandleAssignmentByID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/submit") {
		handler.submitAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/grade") {
		handler.gradeAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/reset") {
		handler.resetAssignment(w, r)
		return
	}

	handler.deleteAssignment(w, r)
}
