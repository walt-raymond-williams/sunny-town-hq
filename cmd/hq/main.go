package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type assignmentResponse struct {
	ID              int64      `json:"id"`
	Category        string     `json:"category"`
	Prompt          string     `json:"prompt"`
	ExpectedAnswer  string     `json:"expected_answer"`
	SubmittedAnswer *string    `json:"submitted_answer"`
	DateSubmitted   time.Time  `json:"date_submitted"`
	Passed          *bool      `json:"passed"`
	Feedback        *string    `json:"feedback"`
	DateGraded      *time.Time `json:"date_graded"`
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

	id := strings.TrimPrefix(r.URL.Path, "/api/assignments/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
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

	var assignment assignmentResponse
	err := app.db.QueryRow(
		r.Context(),
		`
			select id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
			from assignment
			where submitted_answer is null
			order by id asc
			limit 1
		`,
	).Scan(
		&assignment.ID,
		&assignment.Category,
		&assignment.Prompt,
		&assignment.ExpectedAnswer,
		&assignment.SubmittedAnswer,
		&assignment.DateSubmitted,
		&assignment.Passed,
		&assignment.Feedback,
		&assignment.DateGraded,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, map[string]any{
				"assignment": nil,
			})
			return
		}

		log.Printf("load next student assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]assignmentResponse{
		"assignment": assignment,
	})
}

func (app *app) handleStudentGradedAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := app.db.Query(
		r.Context(),
		`
			select id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
			from assignment
			where submitted_answer is not null
				and passed is not null
			order by category asc, date_graded desc, id desc
		`,
	)
	if err != nil {
		log.Printf("list student graded assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "graded assignments could not be loaded",
		})
		return
	}
	defer rows.Close()

	assignments, err := scanAssignments(rows)
	if err != nil {
		log.Printf("scan student graded assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "graded assignments could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) submitAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idText := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/assignments/"), "/submit")
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
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

	var assignment assignmentResponse
	err = app.db.QueryRow(
		r.Context(),
		`
			update assignment
			set submitted_answer = $1
			where id = $2 and submitted_answer is null
			returning id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
		`,
		request.SubmittedAnswer,
		id,
	).Scan(
		&assignment.ID,
		&assignment.Category,
		&assignment.Prompt,
		&assignment.ExpectedAnswer,
		&assignment.SubmittedAnswer,
		&assignment.DateSubmitted,
		&assignment.Passed,
		&assignment.Feedback,
		&assignment.DateGraded,
	)
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

	rows, err := app.db.Query(
		r.Context(),
		`
			select id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
			from assignment
			where submitted_answer is not null
			order by
				case when passed is null then 0 else 1 end,
				date_submitted desc,
				id desc
		`,
	)
	if err != nil {
		log.Printf("list answered assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answered assignments could not be loaded",
		})
		return
	}
	defer rows.Close()

	assignments, err := scanAssignments(rows)
	if err != nil {
		log.Printf("scan answered assignments: %v", err)
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

	idText := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/assignments/"), "/grade")
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
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

	var assignment assignmentResponse
	err = app.db.QueryRow(
		r.Context(),
		`
			update assignment
			set passed = $1,
				feedback = nullif($2, ''),
				date_graded = now()
			where id = $3 and submitted_answer is not null
			returning id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
		`,
		*request.Passed,
		request.Feedback,
		id,
	).Scan(
		&assignment.ID,
		&assignment.Category,
		&assignment.Prompt,
		&assignment.ExpectedAnswer,
		&assignment.SubmittedAnswer,
		&assignment.DateSubmitted,
		&assignment.Passed,
		&assignment.Feedback,
		&assignment.DateGraded,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "answered assignment not found",
			})
			return
		}

		log.Printf("grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
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

	idText := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/assignments/"), "/reset")
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	var assignment assignmentResponse
	err = app.db.QueryRow(
		r.Context(),
		`
			update assignment
			set submitted_answer = null,
				passed = null,
				date_graded = null
			where id = $1
			returning id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
		`,
		id,
	).Scan(
		&assignment.ID,
		&assignment.Category,
		&assignment.Prompt,
		&assignment.ExpectedAnswer,
		&assignment.SubmittedAnswer,
		&assignment.DateSubmitted,
		&assignment.Passed,
		&assignment.Feedback,
		&assignment.DateGraded,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment not found",
			})
			return
		}

		log.Printf("reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be reset",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) listAssignments(w http.ResponseWriter, r *http.Request) {
	rows, err := app.db.Query(
		r.Context(),
		`
			select id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
			from assignment
			order by id desc
		`,
	)
	if err != nil {
		log.Printf("list assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignments could not be loaded",
		})
		return
	}
	defer rows.Close()

	assignments, err := scanAssignments(rows)
	if err != nil {
		log.Printf("scan assignments: %v", err)
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

	var assignment assignmentResponse
	err := app.db.QueryRow(
		r.Context(),
		`
			insert into assignment (category, prompt, expected_answer)
			values ($1, $2, $3)
			returning id, category, prompt, expected_answer, submitted_answer, date_submitted, passed, feedback, date_graded
		`,
		request.Category,
		request.Prompt,
		request.ExpectedAnswer,
	).Scan(
		&assignment.ID,
		&assignment.Category,
		&assignment.Prompt,
		&assignment.ExpectedAnswer,
		&assignment.SubmittedAnswer,
		&assignment.DateSubmitted,
		&assignment.Passed,
		&assignment.Feedback,
		&assignment.DateGraded,
	)
	if err != nil {
		log.Printf("insert assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be saved",
		})
		return
	}

	writeJSON(w, http.StatusCreated, assignment)
}

func scanAssignments(rows pgx.Rows) ([]assignmentResponse, error) {
	assignments := []assignmentResponse{}
	for rows.Next() {
		var assignment assignmentResponse
		if err := rows.Scan(
			&assignment.ID,
			&assignment.Category,
			&assignment.Prompt,
			&assignment.ExpectedAnswer,
			&assignment.SubmittedAnswer,
			&assignment.DateSubmitted,
			&assignment.Passed,
			&assignment.Feedback,
			&assignment.DateGraded,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assignments, nil
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
