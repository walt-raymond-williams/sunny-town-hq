# HQ Architecture and Design

HQ is a small home-network web app for assigning and reviewing simple homework. It is intended to run from a laptop and be usable from that laptop's browser or from a phone/tablet connected to the same Wi-Fi network.

The first version should stay intentionally small:

- A teacher logs in, creates text-prompt assignments, reviews submissions, and marks them pass/fail.
- A student logs in, sees assigned homework, types answers into a text area, and submits them.
- Results are stored in PostgreSQL.
- The Go backend owns authentication, authorization, validation, and business rules.

## Goals

- Learn Go backend development in a realistic but manageable app.
- Learn simple Go API design in a realistic but manageable app.
- Learn PostgreSQL schema design, migrations, and persistence.
- Build a website that works well on laptop and phone browsers.
- Keep local home-network hosting simple enough to actually use.

## Non-Goals for Version 1

- No internet/public hosting.
- No OAuth or third-party login.
- No file uploads.
- No multi-family/multi-school tenancy.
- No complex grading rubrics.
- No real-time chat, notifications, or live collaboration.
- No production-grade operations setup.

## Recommended Stack

- Backend: Go
- Frontend: Vue 3, Vuetify, Vite
- API style: JSON over HTTP
- Database: PostgreSQL
- Optional database REST layer: PostgREST
- Local development database: PostgreSQL container managed by Docker Compose
- Final local hosting: Go serves the built Vue app and API from one port

## High-Level Architecture

```text
Phone / Laptop Browser
  Vue 3 frontend
  JSON API client

        |
        v

Go Backend
  static file server for built Vue app
  HTTP API
  authentication/session handling
  role checks
  assignment/submission business logic

        |
        v

PostgreSQL
  users
  assignments
  submissions

Optional:
PostgREST
  REST access to selected database tables/views for learning and admin exploration
```

The core application path should be:

```text
Vue/Vuetify -> Go -> PostgreSQL
```

PostgREST can run beside the Go app, but it should not be the main API for the student and teacher workflows in version 1. That keeps the Go server meaningful and lets it enforce the rules of the application.

## Local Runtime Decisions

The first version should split local runtime concerns this way:

- Run the Go server directly on the laptop while developing.
- Run PostgreSQL from an official Docker image using Docker Compose.
- Add PostgREST to Docker Compose later if it still feels useful.
- Containerize the Go app later only if portability, repeatable startup, or deployment practice becomes more valuable than the extra Docker setup.

This keeps the early feedback loop simple. The Go server can be started, stopped, debugged, and edited without rebuilding a container, while the database remains portable and easy to reset.

HQ should not use the PostgreSQL install on the host machine. The checked-in project setup should use Docker Compose so the database version, port, credentials, and startup steps are explicit.

Chosen setup:

```text
Go:        run locally
Postgres: run in Docker Compose on localhost:55432
PostgREST: optional Docker Compose sidecar later
```

## Why Go Should Serve the Frontend

Vue's production build is static HTML, CSS, and JavaScript. Go can serve those files easily.

For this app, using Go as the single web entry point is a good fit because:

- There is only one app.
- The app is intended for a home network.
- Other devices only need one URL.
- The backend and frontend can be deployed together.
- It avoids adding Nginx before it is actually useful.

Development mode can still use separate servers:

```text
Vue dev server: http://localhost:5173
Go API server:  http://localhost:8080
```

Home-network mode should use one server:

```text
Go server: http://<laptop-ip>:8080
```

The Go server should expose:

```text
/              built Vue/Vuetify app
/assets/...    Vue static assets
/api/...       JSON API
/healthz       health check
```

## Current Stage 1 Demo

The current demo has already moved from plain HTML to a Vue 3 frontend with Vuetify components.

Current pieces:

- `frontend/` contains the Vue 3 + Vuetify source app.
- `frontend/vite.config.js` builds the frontend into `web/`.
- `web/` contains generated production assets served by Go.
- Go serves the frontend and exposes `/api/...`.
- PostgreSQL runs from Docker Compose on `localhost:55432`.

Current implemented screens:

- Splash page with Student and Teacher choices.
- Student page that shows one unanswered assignment at a time.
- Student answering page displays teacher feedback on reset assignments.
- Student answer submission that advances to the next unanswered assignment.
- Student completion message when all assignments have submitted answers.
- Student grades tab with category pass percentages and graded-question lists.
- Teacher password gate using the hard-coded password `local-demo-password`.
- Teacher tabs for grading and assignment creation.
- Assignment creation with category, prompt, and expected answer.
- Assignment list with expandable answers and delete action.
- Grading list for submitted answers.
- Inline pass/fail and feedback form for answered assignments.
- Teacher reset action that clears submitted answer and pass/fail result while keeping feedback.

Current implemented API:

```text
POST   /api/teacher/login
GET    /api/student/assignments/next
GET    /api/student/assignments/graded
GET    /api/assignments
GET    /api/assignments/answered
POST   /api/assignments
POST   /api/assignments/:id/submit
PATCH  /api/assignments/:id/grade
PATCH  /api/assignments/:id/reset
DELETE /api/assignments/:id
GET    /healthz
```

Current demo table:

```text
assignment
  id
  category
  prompt
  expected_answer
  submitted_answer
  date_submitted
  passed
  feedback
  date_graded
```

## Users and Roles

Version 1 has two roles.

### Student

The student can:

- Log in.
- See assigned homework.
- Open an assignment.
- Type an answer into a text area.
- Submit the answer.
- See pass/fail result and feedback after the teacher reviews it.

### Teacher

The teacher can:

- Log in.
- Create assignments.
- View all assignments.
- View student submissions.
- Mark submissions pass/fail.
- Leave feedback.

## First User Flows

### Student Flow

```text
Student logs in
  -> sees assignment list
  -> opens assignment
  -> reads prompt
  -> types answer
  -> submits answer
  -> sees submitted status
```

### Teacher Flow

```text
Teacher logs in
  -> creates assignment
  -> sees submissions
  -> opens submitted answer
  -> enters pass/fail result and feedback
  -> marks submission reviewed
```

These two flows are the first real product slice. If they work, the app is useful.

## API Design

The first version should use a simple JSON-over-HTTP API. This keeps the app easy to build, test, and understand while still giving the Go backend full ownership of authentication, authorization, validation, and business rules.

Proposed endpoints:

```text
POST /api/login
POST /api/logout
GET  /api/me

GET  /api/assignments
GET  /api/assignments/:id
POST /api/assignments

POST /api/assignments/:id/submissions
GET  /api/submissions
PATCH /api/submissions/:id/grade
```

Endpoint behavior:

```text
POST /api/login
  Accepts username and password.
  Creates a session cookie.

GET /api/me
  Returns the current logged-in user.

GET /api/assignments
  For students, returns available homework.
  For teachers, returns assignments they can review/manage.

POST /api/assignments
  Teacher-only.
  Creates a text-prompt assignment.

POST /api/assignments/:id/submissions
  Student-only.
  Submits a text answer for an assignment.

GET /api/submissions
  Teacher-only.
  Lists submitted answers for review.

PATCH /api/submissions/:id/grade
  Teacher-only.
  Adds grade and feedback.
```

The frontend should keep API calls in a small `src/api/` module so views do not need to know fetch details.

## Database Design

Initial tables:

```text
users
  id
  username
  password_hash
  role
  display_name
  created_at

assignments
  id
  title
  prompt
  created_by_user_id
  due_at
  created_at
  archived_at

submissions
  id
  assignment_id
  student_user_id
  answer_text
  status
  submitted_at
  graded_at
  grade
  feedback
  graded_by_user_id
```

Recommended first statuses:

```text
submitted
graded
```

Drafts can be added later if needed.

## Authentication and Authorization

Use simple username/password login for version 1.

Recommended approach:

- Store password hashes, never plain text passwords.
- Use cookie-based sessions.
- Add a `Me` endpoint so the frontend can discover the current user.
- Protect teacher-only actions on the backend.
- Protect student-only actions on the backend.

Frontend route guards are useful for user experience, but the backend must enforce the real permissions.

## PostgREST Role

PostgREST makes sense as a learning and inspection tool.

Good uses:

- Explore how database tables map to REST endpoints.
- Inspect data while developing.
- Build limited admin views later.
- Compare direct database REST access with the app-specific Go API.

Avoid using PostgREST as the main student/teacher API in version 1. If Vue talks directly to PostgREST, it becomes harder to keep business rules centralized in Go.

## Local Network Hosting

The app should run on the laptop and listen on all network interfaces:

```text
0.0.0.0:8080
```

Other devices on the same Wi-Fi can access it with:

```text
http://<laptop-ip>:8080
```

The laptop's IP can be found with:

```powershell
ipconfig
```

Look for the Wi-Fi adapter's IPv4 address.

For reliable use:

- Allow the Go server through Windows Firewall on private networks.
- Keep the laptop awake while the site is in use.
- Consider reserving the laptop's IP address in the router settings.

## Suggested Repository Layout

```text
.
  backend/
    cmd/
      hq/
        main.go
    internal/
      auth/
      db/
      assignments/
      submissions/
      http/
    migrations/

  frontend/
    src/
      App.vue
      main.js
      style.css
    package.json
    vite.config.js

  deploy/
    docker-compose.yml
    postgrest.conf

  docs/
    decisions.md

  ARCHITECTURE.md
  README.md
```

## Build Milestones

### Milestone 1: Project Foundation

- Create Go backend skeleton.
- Create Vue 3 frontend.
- Add Docker Compose for PostgreSQL using an official Postgres image.
- Add database migrations.
- Add seed teacher and student users.

### Milestone 2: API Foundation

- Add JSON API routes.
- Add health check.
- Add database connection.

### Milestone 3: Authentication

- Add login/logout.
- Add session cookie support.
- Add `Me` endpoint.
- Add role checks.

### Milestone 4: Student Workflow

- Student login screen.
- Assignment list.
- Assignment detail page.
- Text answer submission.

### Milestone 5: Teacher Workflow

- Teacher assignment creation.
- Submission review page.
- Grade and feedback form.

### Milestone 6: Home-Network Mode

- Build the Vue frontend.
- Serve the built frontend from Go.
- Bind Go server to `0.0.0.0:8080`.
- Test from laptop browser and phone browser.

### Milestone 7: PostgREST Add-On

- Add PostgREST to Docker Compose.
- Connect it to PostgreSQL.
- Expose limited tables or views.
- Document example PostgREST queries.

## Initial Design Decisions

Recommended decisions for version 1:

- Use one Git repository.
- Use Go as the main backend and final static file server.
- Use Vue 3 with Vuetify for the frontend.
- Use JSON over HTTP for the app API.
- Use PostgreSQL directly from Go.
- Use Docker Compose for the default PostgreSQL development database.
- Run Go locally during early development.
- Delay containerizing the Go app until the project has enough shape to benefit from it.
- Include PostgREST as a sidecar, not the primary app API.
- Use cookie sessions for login.
- Start with one teacher and one student.
- Start with text-only assignments and text-only submissions.

## Future Ideas

- Multiple students.
- Assignment due dates and late status.
- Draft answers.
- File attachments.
- Multiple-choice questions.
- Subject/category labels.
- Grade history.
- Parent dashboard.
- Printable assignments.
- HTTPS with a local reverse proxy.
- Nginx in front of Go for production-style practice.
