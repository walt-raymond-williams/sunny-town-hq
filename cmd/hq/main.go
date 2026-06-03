package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db *pgxpool.Pool
}

type createAssignmentRequest struct {
	Category       string `json:"category"`
	Prompt         string `json:"prompt"`
	ExpectedAnswer string `json:"expected_answer"`
}

type teacherLoginRequest struct {
	Password string `json:"password"`
}

type submitAssignmentRequest struct {
	SubmittedAnswer string `json:"submitted_answer"`
}

type gradeAssignmentRequest struct {
	Passed   *bool  `json:"passed"`
	Feedback string `json:"feedback"`
}

type resetAssignmentRequest struct {
	Feedback string `json:"feedback"`
}

type assignmentAttemptResponse struct {
	ID              int64      `json:"id"`
	AssignmentID    int64      `json:"assignment_id"`
	AttemptNumber   int        `json:"attempt_number"`
	SubmittedAnswer string     `json:"submitted_answer"`
	DateSubmitted   time.Time  `json:"date_submitted"`
	Passed          *bool      `json:"passed"`
	Feedback        *string    `json:"feedback"`
	DateGraded      *time.Time `json:"date_graded"`
	ResetAt         *time.Time `json:"reset_at"`
}

type assignmentResponse struct {
	ID             int64                       `json:"id"`
	Category       string                      `json:"category"`
	Prompt         string                      `json:"prompt"`
	ExpectedAnswer string                      `json:"expected_answer"`
	CreatedAt      time.Time                   `json:"created_at"`
	CurrentAttempt *assignmentAttemptResponse  `json:"current_attempt"`
	Attempts       []assignmentAttemptResponse `json:"attempts"`
}

func main() {
	ctx := context.Background()
	host := envOrDefault("HQ_HOST", "0.0.0.0")
	port := envOrDefault("HQ_PORT", "8080")
	addr := host + ":" + port
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required, for example: postgres://hq:hq@localhost:55432/hq?sslmode=disable")
	}

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	app := &app{db: db}

	webRoot := filepath.Join(".", "web")
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/teacher/login", app.handleTeacherLogin)
	mux.HandleFunc("/api/teacher/logout", app.handleTeacherLogout)
	mux.HandleFunc("/api/student/assignments/next", app.handleNextStudentAssignment)
	mux.HandleFunc("/api/student/assignments/graded", app.handleStudentGradedAssignments)
	mux.HandleFunc("/api/assignments/answered", app.handleAnsweredAssignments)
	mux.HandleFunc("/api/assignments", app.handleAssignments)
	mux.HandleFunc("/api/assignments/", app.handleAssignmentByID)

	mux.HandleFunc("/", staticHandler(webRoot))

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HQ server listening on http://%s", addr)
	for _, ip := range localIPv4Addresses() {
		log.Printf("Try from another device on Wi-Fi: http://%s:%s", ip, port)
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func (app *app) handleTeacherLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request teacherLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	if request.Password != "local-demo-password" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "incorrect teacher password",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "hq_teacher",
		Value:    "local-demo-password",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 8,
	})
	writeJSON(w, http.StatusOK, map[string]string{
		"role": "teacher",
	})
}

func (app *app) handleTeacherLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "hq_teacher",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "logged out",
	})
}

func (app *app) handleAssignments(w http.ResponseWriter, r *http.Request) {
	if !isTeacher(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "teacher login required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		app.listAssignments(w, r)
	case http.MethodPost:
		app.createAssignment(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *app) handleAssignmentByID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/submit") {
		app.submitAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/grade") {
		app.gradeAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/reset") {
		app.resetAssignment(w, r)
		return
	}

	if !isTeacher(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "teacher login required",
		})
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "")
	if !ok {
		return
	}

	result, err := app.db.Exec(r.Context(), "delete from assignment where id = $1", id)
	if err != nil {
		log.Printf("delete assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be deleted",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *app) handleNextStudentAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		`
			where not exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
			)
			order by a.id asc
			limit 1
		`,
	)
	if err != nil {
		log.Printf("load next student assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be loaded",
		})
		return
	}

	if len(assignments) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"assignment": nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]assignmentResponse{
		"assignment": assignments[0],
	})
}

func (app *app) handleStudentGradedAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		`
			where exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.passed is not null
			)
			order by a.category asc, a.id desc
		`,
	)
	if err != nil {
		log.Printf("list student graded assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "graded assignments could not be loaded",
		})
		return
	}

	for index := range assignments {
		assignments[index].Attempts = gradedAttempts(assignments[index].Attempts)
		assignments[index].CurrentAttempt = currentAttempt(assignments[index].Attempts)
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) submitAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/submit")
	if !ok {
		return
	}

	var request submitAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.SubmittedAnswer = strings.TrimSpace(request.SubmittedAnswer)
	if request.SubmittedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "submitted_answer is required",
		})
		return
	}

	var attemptID int64
	err := app.db.QueryRow(
		r.Context(),
		`
			insert into assignment_attempt (assignment_id, attempt_number, submitted_answer)
			select a.id,
				coalesce(max(aa.attempt_number), 0) + 1,
				$1
			from assignment a
			left join assignment_attempt aa on aa.assignment_id = a.id
			where a.id = $2
				and not exists (
					select 1
					from assignment_attempt active_attempt
					where active_attempt.assignment_id = a.id
						and active_attempt.reset_at is null
				)
			group by a.id
			returning id
		`,
		request.SubmittedAnswer,
		id,
	).Scan(&attemptID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment not found or already submitted",
			})
			return
		}

		log.Printf("submit assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answer could not be submitted",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load submitted assignment %d after attempt %d: %v", id, attemptID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answer was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) handleAnsweredAssignments(w http.ResponseWriter, r *http.Request) {
	if !isTeacher(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "teacher login required",
		})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		`
			where exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
			)
			order by (
				select case when aa.passed is null then 0 else 1 end
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			), (
				select aa.date_submitted
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			) desc, a.id desc
		`,
	)
	if err != nil {
		log.Printf("list answered assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answered assignments could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) gradeAssignment(w http.ResponseWriter, r *http.Request) {
	if !isTeacher(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "teacher login required",
		})
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/grade")
	if !ok {
		return
	}

	var request gradeAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Feedback = strings.TrimSpace(request.Feedback)
	if request.Passed == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "passed is required",
		})
		return
	}

	result, err := app.db.Exec(
		r.Context(),
		`
			update assignment_attempt
			set passed = $1,
				feedback = nullif($2, ''),
				date_graded = now()
			where id = (
				select aa.id
				from assignment_attempt aa
				where aa.assignment_id = $3
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			)
		`,
		*request.Passed,
		request.Feedback,
		id,
	)
	if err != nil {
		log.Printf("grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load graded assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) resetAssignment(w http.ResponseWriter, r *http.Request) {
	if !isTeacher(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "teacher login required",
		})
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/reset")
	if !ok {
		return
	}

	var request resetAssignmentRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}
	}
	request.Feedback = strings.TrimSpace(request.Feedback)

	result, err := app.db.Exec(
		r.Context(),
		`
			update assignment_attempt
			set feedback = nullif($1, ''),
				reset_at = now()
			where id = (
				select aa.id
				from assignment_attempt aa
				where aa.assignment_id = $2
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			)
		`,
		request.Feedback,
		id,
	)
	if err != nil {
		log.Printf("reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be reset",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment was reset but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) listAssignments(w http.ResponseWriter, r *http.Request) {
	assignments, err := app.loadAssignments(r.Context(), "order by a.id desc")
	if err != nil {
		log.Printf("list assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignments could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) createAssignment(w http.ResponseWriter, r *http.Request) {
	var request createAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Category = strings.ToUpper(strings.TrimSpace(request.Category))
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.ExpectedAnswer = strings.TrimSpace(request.ExpectedAnswer)

	if !isValidCategory(request.Category) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "category must be MATH, SCIENCE, or READING",
		})
		return
	}

	if request.Prompt == "" || request.ExpectedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "prompt and expected_answer are required",
		})
		return
	}

	var id int64
	err := app.db.QueryRow(
		r.Context(),
		`
			insert into assignment (category, prompt, expected_answer)
			values ($1, $2, $3)
			returning id
		`,
		request.Category,
		request.Prompt,
		request.ExpectedAnswer,
	).Scan(&id)
	if err != nil {
		log.Printf("insert assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be saved",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load created assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusCreated, assignment)
}

func (app *app) loadAssignmentByID(ctx context.Context, id int64) (assignmentResponse, error) {
	assignments, err := app.loadAssignments(ctx, "where a.id = $1", id)
	if err != nil {
		return assignmentResponse{}, err
	}

	if len(assignments) == 0 {
		return assignmentResponse{}, pgx.ErrNoRows
	}

	return assignments[0], nil
}

func (app *app) loadAssignments(ctx context.Context, suffix string, args ...any) ([]assignmentResponse, error) {
	query := `
		select a.id, a.category, a.prompt, a.expected_answer, a.created_at
		from assignment a
		` + suffix

	rows, err := app.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := []assignmentResponse{}
	assignmentIndexes := map[int64]int{}
	for rows.Next() {
		var assignment assignmentResponse
		if err := rows.Scan(
			&assignment.ID,
			&assignment.Category,
			&assignment.Prompt,
			&assignment.ExpectedAnswer,
			&assignment.CreatedAt,
		); err != nil {
			return nil, err
		}

		assignment.Attempts = []assignmentAttemptResponse{}
		assignmentIndexes[assignment.ID] = len(assignments)
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(assignments) == 0 {
		return assignments, nil
	}

	ids := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ID)
	}

	attemptRows, err := app.db.Query(
		ctx,
		`
			select id, assignment_id, attempt_number, submitted_answer, date_submitted, passed, feedback, date_graded, reset_at
			from assignment_attempt
			where assignment_id = any($1)
			order by assignment_id asc, attempt_number asc
		`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer attemptRows.Close()

	for attemptRows.Next() {
		var attempt assignmentAttemptResponse
		if err := attemptRows.Scan(
			&attempt.ID,
			&attempt.AssignmentID,
			&attempt.AttemptNumber,
			&attempt.SubmittedAnswer,
			&attempt.DateSubmitted,
			&attempt.Passed,
			&attempt.Feedback,
			&attempt.DateGraded,
			&attempt.ResetAt,
		); err != nil {
			return nil, err
		}

		index, ok := assignmentIndexes[attempt.AssignmentID]
		if !ok {
			continue
		}
		assignments[index].Attempts = append(assignments[index].Attempts, attempt)
	}

	if err := attemptRows.Err(); err != nil {
		return nil, err
	}

	for index := range assignments {
		assignments[index].CurrentAttempt = currentAttempt(assignments[index].Attempts)
	}

	return assignments, nil
}

func currentAttempt(attempts []assignmentAttemptResponse) *assignmentAttemptResponse {
	for index := len(attempts) - 1; index >= 0; index-- {
		if attempts[index].ResetAt == nil {
			return &attempts[index]
		}
	}

	return nil
}

func gradedAttempts(attempts []assignmentAttemptResponse) []assignmentAttemptResponse {
	graded := []assignmentAttemptResponse{}
	for _, attempt := range attempts {
		if attempt.Passed != nil {
			graded = append(graded, attempt)
		}
	}

	return graded
}

func parseAssignmentID(w http.ResponseWriter, path string, suffix string) (int64, bool) {
	idText := strings.TrimPrefix(path, "/api/assignments/")
	if suffix != "" {
		idText = strings.TrimSuffix(idText, suffix)
	}
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	return id, true
}

func isTeacher(r *http.Request) bool {
	cookie, err := r.Cookie("hq_teacher")
	return err == nil && cookie.Value == "local-demo-password"
}

func isValidCategory(category string) bool {
	return category == "MATH" || category == "SCIENCE" || category == "READING"
}

func staticHandler(webRoot string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(webRoot))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := filepath.Clean(r.URL.Path)
		if path == "." || path == string(filepath.Separator) {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		fullPath := filepath.Join(webRoot, strings.TrimPrefix(path, string(filepath.Separator)))
		if _, err := os.Stat(fullPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(webRoot, "index.html"))
	}
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
