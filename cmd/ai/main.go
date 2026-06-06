package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"hq/internal/aiapi"
	"hq/internal/serviceauth"
)

type app struct {
	hqBaseURL           string
	aiToHQSecret        string
	aiServiceSecret     string
	provider            string
	model               string
	graderPromptVersion string
	client              *http.Client
}

const (
	aiGradeStatusCompleted = "completed"
	aiGradeStatusFailed    = "failed"
)

func main() {
	host := envOrDefault("AI_HOST", "0.0.0.0")
	port := envOrDefault("AI_PORT", "18083")
	addr := host + ":" + port

	app := &app{
		hqBaseURL:           strings.TrimRight(envOrDefault("HQ_INTERNAL_BASE_URL", "http://127.0.0.1:18080"), "/"),
		aiToHQSecret:        envOrDefault("AI_TO_HQ_SERVICE_SECRET", "local-dev-ai-service-secret"),
		aiServiceSecret:     envOrDefault("HQ_TO_AI_SERVICE_SECRET", "local-dev-hq-to-ai-secret"),
		provider:            envOrDefault("AI_PROVIDER", "fake"),
		model:               envOrDefault("OPENAI_MODEL", "fake-grader"),
		graderPromptVersion: envOrDefault("AI_PROMPT_VERSION_GRADER", "assignment-grader-v1"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/internal/ai/grade-assignment", app.handleGradeAssignment)

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("AI service listening on http://%s", addr)
	for _, ip := range localIPv4Addresses() {
		log.Printf("AI service local network address: http://%s:%s", ip, port)
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func (app *app) handleGradeAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !serviceauth.Authorized(r.Header, serviceauth.AIServiceSecret, app.aiServiceSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	var request aiapi.GradeAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}
	request.RequestID = strings.TrimSpace(request.RequestID)
	if request.RequestID == "" || request.AttemptID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request_id and attempt_id are required",
		})
		return
	}

	status := aiGradeStatusCompleted
	if err := app.gradeAssignment(r.Context(), request); err != nil {
		status = aiGradeStatusFailed
		log.Printf("grade assignment attempt %d: %v", request.AttemptID, err)
	}

	writeJSON(w, http.StatusOK, aiapi.GradeAssignmentResponse{
		RequestID: request.RequestID,
		Status:    status,
	})
}

func (app *app) gradeAssignment(ctx context.Context, request aiapi.GradeAssignmentRequest) error {
	contextResponse, err := app.loadGradingContext(ctx, request.AttemptID)
	if err != nil {
		return err
	}

	result := app.fakeGrade(request, contextResponse)
	if app.provider != "fake" {
		failed := aiapi.AIGradeResultRequest{
			RequestID:     request.RequestID,
			Status:        aiGradeStatusFailed,
			PromptVersion: app.graderPromptVersion,
			Model:         app.model,
			ErrorMessage:  "AI provider is not implemented; set AI_PROVIDER=fake until OpenAI support is added",
		}
		if postErr := app.postAIGradeResult(ctx, request.AttemptID, failed); postErr != nil {
			return postErr
		}
		return errors.New("ai provider is not implemented")
	}

	return app.postAIGradeResult(ctx, request.AttemptID, result)
}

func (app *app) fakeGrade(request aiapi.GradeAssignmentRequest, contextResponse aiapi.GradingContextResponse) aiapi.AIGradeResultRequest {
	passed := normalizeAnswer(contextResponse.SubmittedAnswer) == normalizeAnswer(contextResponse.ExpectedAnswer)
	confidence := 0.95
	feedback := "The answer matches the expected answer."
	score := 1.0
	if !passed {
		feedback = "The answer does not match the expected answer."
		score = 0
		confidence = 0.8
	}

	return aiapi.AIGradeResultRequest{
		RequestID:           request.RequestID,
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: feedback,
		Confidence:          &confidence,
		RubricScores: []aiapi.RubricScore{{
			Name:   "expected_answer_match",
			Score:  score,
			Reason: "Fake grader compares normalized submitted and expected answers.",
		}},
		Model:         app.model,
		PromptVersion: app.graderPromptVersion,
		RawResponse: map[string]any{
			"provider": "fake",
		},
	}
}

func (app *app) loadGradingContext(ctx context.Context, attemptID int64) (aiapi.GradingContextResponse, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		app.hqBaseURL+"/api/internal/ai/assignment-attempts/"+strconv.FormatInt(attemptID, 10)+"/grading-context",
		nil,
	)
	if err != nil {
		return aiapi.GradingContextResponse{}, err
	}
	request.Header.Set(serviceauth.HQServiceNameHeader, "ai")
	request.Header.Set(serviceauth.HQServiceSecret, app.aiToHQSecret)

	response, err := app.client.Do(request)
	if err != nil {
		return aiapi.GradingContextResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return aiapi.GradingContextResponse{}, fmt.Errorf("hq grading context returned status %d", response.StatusCode)
	}

	var contextResponse aiapi.GradingContextResponse
	if err := json.NewDecoder(response.Body).Decode(&contextResponse); err != nil {
		return aiapi.GradingContextResponse{}, err
	}
	return contextResponse, nil
}

func (app *app) postAIGradeResult(ctx context.Context, attemptID int64, result aiapi.AIGradeResultRequest) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		app.hqBaseURL+"/api/internal/ai/assignment-attempts/"+strconv.FormatInt(attemptID, 10)+"/ai-grade",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(serviceauth.HQServiceNameHeader, "ai")
	request.Header.Set(serviceauth.HQServiceSecret, app.aiToHQSecret)

	response, err := app.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("hq ai grade result returned status %d", response.StatusCode)
	}
	return nil
}

func normalizeAnswer(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func localIPv4Addresses() []string {
	var addresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		ifaceAddresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, ifaceAddress := range ifaceAddresses {
			ipNet, ok := ifaceAddress.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			addresses = append(addresses, fmt.Sprintf("%s", ip))
		}
	}

	return addresses
}
