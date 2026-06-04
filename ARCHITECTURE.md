# HQ Architecture and Design

HQ is a small home-network web app for assigning and reviewing simple homework. It is intended to run from a laptop and be usable from that laptop's browser or from a phone/tablet connected to the same Wi-Fi network.

The first version should stay intentionally small:

- A teacher logs in, creates text-prompt assignments, reviews submissions, and marks them pass/fail.
- A student enters without credentials, sees assigned homework, types answers into a text area, and submits them.
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
- API style: JSON over HTTP for homework workflows, Connect RPC for the pet workflow
- Database: PostgreSQL
- Optional database REST layer: PostgREST
- Local development database: PostgreSQL container managed by Docker Compose
- Final local hosting: Go serves the built Vue app and API from one port

## High-Level Architecture

```text
Phone / Laptop Browser
  Vue 3 frontend
  JSON API client
  Connect RPC pet client

        |
        v

Go Backend
  static file server for built Vue app
  HTTP API
  Connect RPC PetService
  authentication/session handling
  role checks
  assignment/submission business logic

        |
        v

PostgreSQL
  app_user
  pet_state
  assignment
  assignment_attempt

Optional:
PostgREST
  REST access to selected database tables/views for learning and admin exploration
```

The core homework application path should be:

```text
Vue/Vuetify -> Go -> PostgreSQL
```

The pet feature uses:

```text
Vue/Pinia generated Connect client -> Go Connect PetService -> PostgreSQL
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
Go:        run locally when Go is installed, or with a disposable Go helper container
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

The current laptop is using a disposable Go helper container because Go is not on PATH. That maps the app's internal `8080` port to host port `18080`, so the current browser URL is:

```text
http://127.0.0.1:18080
```

Other devices on the same Wi-Fi should use the laptop IP with that mapped port:

```text
http://<laptop-ip>:18080
```

The Go server should expose:

```text
/              built Vue/Vuetify app
/assets/...    Vue static assets
/api/...       JSON API
/hq.pet.v1.PetService/... Connect RPC pet API
/healthz       health check
```

## Current Stage 1 Demo

The current demo has already moved from plain HTML to a Vue 3 frontend with Vuetify components.

Current pieces:

- `frontend/` contains the Vue 3 + Vuetify source app.
- `frontend/src/router.js` defines Vue Router routes so browser back/forward works between app pages.
- `frontend/vite.config.js` builds the frontend into `web/`.
- `web/` contains generated production assets served by Go.
- Go serves the frontend and exposes `/api/...` plus `/hq.pet.v1.PetService/...`.
- PostgreSQL runs from Docker Compose on `localhost:55432`.

Current implemented screens:

- Splash page with Student and Teacher choices.
- Student page that shows one unanswered assignment at a time, with an optional subject filter for All, Math, Science, or Reading.
- Student answering page displays previous attempts below the answer form as an expandable list.
- Student answer submission that advances to the next unanswered assignment.
- Student completion message when all assignments have submitted answers.
- Student grades tab with category pass percentages and graded-question lists grouped by question, with top-level pass/fail indicators.
- Student Pet tab with cookie count, hunger, happiness, and energy.
- Student feed action that spends one cookie and increases hunger by 10.
- Student virtual pet avatar that animates on the Student page.
- Teacher password gate using the hard-coded password `local-demo-password`.
- Teacher logout that clears the teacher cookie and returns to the splash page.
- Unified Teacher Desk questions workspace with creation, filters, grading, reset, history, and delete.
- Collapsible assignment creation with category, prompt, and expected answer.
- Assignment list with `Needs Review`, `All`, `Unanswered`, `Reset`, and `Graded` filters.
- Inline pass/fail, feedback, reset, attempt history, and delete actions from each expanded question.
- Teacher reset action that saves feedback, marks the active attempt reset, and preserves attempt history.

Current frontend routes:

```text
/                Splash page
/student         Student workspace
/teacher/login   Teacher password page
/teacher         Teacher Desk
```

Current implemented API:

```text
POST   /api/teacher/login
POST   /api/teacher/logout
GET    /api/student/assignments/next
GET    /api/student/assignments/graded
GET    /api/student/profile
POST   /api/student/pet/feed
POST   /hq.pet.v1.PetService/GetPetState
POST   /hq.pet.v1.PetService/FeedPet
POST   /hq.pet.v1.PetService/PlayWithPet
POST   /hq.pet.v1.PetService/PutPetToSleep
POST   /hq.pet.v1.PetService/WakePet
POST   /hq.pet.v1.PetService/WatchPetState
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
  created_at

assignment_attempt
  id
  assignment_id
  attempt_number
  submitted_answer
  date_submitted
  passed
  feedback
  date_graded
  cookie_awarded
  reset_at

app_user
  id
  display_name
  cookies

pet_state
  id
  user_id
  hunger
  happiness
  energy
  sleeping
  sleep_started_at
  sleep_started_energy
  updated_at
  last_decay_at
```

## Current Users and Roles

Version 1 has two roles.

### Student

The student can:

- Enter without credentials from the splash page.
- See one unanswered assignment at a time.
- Type an answer into a text area.
- Submit the answer.
- See previous attempts when a reset assignment is shown again.
- View graded attempts grouped by category and question.
- See category pass percentages for Math, Science, and Reading.
- See the current cookie count and pet stats on the Pet tab.
- Feed the pet one cookie at a time.

### Teacher

The teacher can:

- Log in with the hard-coded demo password `local-demo-password`.
- Create assignments.
- View all assignments.
- View student submissions.
- Mark submissions pass/fail.
- Leave feedback.
- Reset a submitted assignment to an unanswered state while keeping the old attempt.
- Delete assignments.

## Current User Flows

### Student Flow

```text
Student chooses Student
  -> sees the next unanswered assignment
  -> reads prompt
  -> reviews prior attempts if the teacher reset the problem
  -> types answer
  -> submits answer
  -> receives the next unanswered assignment
  -> sees completion message when no unanswered work remains
```

Student grade review:

```text
Student chooses View Grades
  -> sees pass percentages by category
  -> reviews graded attempts grouped by Math, Science, and Reading
```

### Teacher Flow

```text
Teacher chooses Teacher
  -> enters hard-coded demo password
  -> lands on the unified questions workspace
  -> creates assignments from the New Question panel
  -> filters questions by review state
  -> opens a question
  -> chooses pass or fail
  -> optionally leaves feedback
  -> saves result
  -> can save feedback and reset the assignment if the student should try again
```

These two flows are the first real product slice. If they work, the app is useful.

## API Design

The first version uses JSON-over-HTTP for homework workflows and Connect RPC for the virtual pet workflow. Connect keeps the pet API protobuf-first and gives the Vue frontend generated TypeScript client types while still working cleanly in the browser.

Implemented endpoints:

```text
POST   /api/teacher/login
POST   /api/teacher/logout
GET    /api/student/assignments/next
GET    /api/student/assignments/graded
GET    /api/student/profile
POST   /api/student/pet/feed
POST   /hq.pet.v1.PetService/GetPetState
POST   /hq.pet.v1.PetService/FeedPet
POST   /hq.pet.v1.PetService/PlayWithPet
POST   /hq.pet.v1.PetService/PutPetToSleep
POST   /hq.pet.v1.PetService/WakePet
POST   /hq.pet.v1.PetService/WatchPetState
GET    /api/assignments
GET    /api/assignments/answered
POST   /api/assignments
POST   /api/assignments/:id/submit
PATCH  /api/assignments/:id/grade
PATCH  /api/assignments/:id/reset
DELETE /api/assignments/:id
GET    /healthz
```

Endpoint behavior:

```text
POST /api/teacher/login
  Accepts the demo teacher password.
  Creates a teacher cookie.

POST /api/teacher/logout
  Clears the teacher cookie.

GET /api/student/assignments/next
  Returns the next unanswered assignment. Optional query: category=MATH, SCIENCE, or READING.

GET /api/student/assignments/graded
  Returns graded assignments for the student grade view.

GET /api/student/profile
  Returns the demo student profile, cookie count, and pet stats.

POST /api/student/pet/feed
  Legacy JSON endpoint.
  Spends one cookie and increases pet hunger by 10, capped at 100.

The current Vue pet UI uses the generated Connect client in `frontend/src/stores/studentPet.js`.
The JSON profile and feed endpoints still exist on the Go server, but they are no longer the main frontend pet path.

PetService Connect RPC
  Protobuf-backed pet state and interactions.
  GetPetState returns cookies, hunger, happiness, energy, sleeping, mood, updated_at, and last_decay_at.
  FeedPet spends one cookie and increases hunger.
  PlayWithPet increases happiness and spends 10 energy.
  PutPetToSleep starts a ten-minute energy recovery.
  WakePet wakes the pet early, stops energy recovery, and subtracts 30 happiness.
  WatchPetState streams current pet state about once per second while the Student page is open.

GET /api/assignments
  Teacher-only.
  Returns all assignments.

GET /api/assignments/answered
  Teacher-only.
  Returns submitted assignments for grading.

POST /api/assignments
  Teacher-only.
  Creates a categorized text-prompt assignment.

POST /api/assignments/:id/submit
  Public student workflow.
  Submits a text answer for an assignment.

PATCH /api/assignments/:id/grade
  Teacher-only.
  Saves pass/fail and optional feedback on the latest active attempt.

PATCH /api/assignments/:id/reset
  Teacher-only.
  Saves optional feedback and marks the latest active attempt reset so the assignment can be answered again.

DELETE /api/assignments/:id
  Teacher-only.
  Deletes an assignment.
```

The current frontend keeps API calls inside `frontend/src/App.vue`. That is acceptable for this early demo, but moving fetch helpers into a small `src/api/` module would be a good cleanup once the app grows.

## Database Design

Current table:

```text
assignment
  id
  category
  prompt
  expected_answer
  created_at

assignment_attempt
  id
  assignment_id
  attempt_number
  submitted_answer
  date_submitted
  passed
  feedback
  date_graded
  cookie_awarded
  reset_at

app_user
  id
  display_name
  cookies

pet_state
  id
  user_id
  hunger
  happiness
  energy
  sleeping
  sleep_started_at
  sleep_started_energy
  updated_at
  last_decay_at
```

Category is constrained to:

```text
MATH
SCIENCE
READING
```

`assignment` stores the stable question. `assignment_attempt` stores each submitted answer cycle. `passed` is nullable on attempts: `null` means the attempt has not been reviewed yet, `true` means passed, and `false` means failed. `reset_at` marks an attempt as historical and makes the assignment answerable again.

The demo seed file at `deploy/postgres/seed/002_second_grade_questions.sql` clears existing assignment records and inserts 20 Math, 20 Science, and 20 Reading questions for a second-grade student.

`app_user` stores the single demo student profile. Its `cookies` value is the future virtual-pet feeding currency. When a teacher grades an attempt as passed, the backend awards one cookie to the student and marks that attempt's `cookie_awarded` flag so saving the same passing result again does not duplicate the reward.

`pet_state` stores the single demo pet's care stats. Hunger, happiness, and energy are integer values from `0` to `100` and start at `50`. The pet also tracks whether it is sleeping, when sleep started, the energy value at sleep start, and when decay was last applied. The Vue frontend loads this data through a Pinia store backed by generated Connect clients.

Pet decay runs two ways: a background ticker applies decay about every five minutes, and each pet RPC lazily catches up from `last_decay_at`. Hunger, happiness, and awake energy decay at about `144` points per day. Sleeping restores energy linearly from `sleep_started_energy` to `100` over ten minutes, then the pet wakes automatically and gains 10 happiness. If the student wakes the pet early, energy recovery stops and happiness drops by 30. If awake energy reaches `0`, the pet falls asleep. The Student page opens the `WatchPetState` stream so the floating avatar and Pet tab stats can update live without polling.

## Authentication and Authorization

The current demo uses one hard-coded teacher password and no student credentials.

Current approach:

- Teacher login accepts `local-demo-password`.
- Successful teacher login sets an HTTP-only cookie.
- Teacher logout clears the teacher cookie.
- Teacher-only endpoints check that cookie.
- Student answering and grade viewing are public inside the home-network app.

Future production-style authentication should store password hashes, use real users, expose a `Me` endpoint, and keep backend role checks as the source of truth.

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
  cmd/
    hq/
      main.go
      pet_service.go

  frontend/
    src/
      components/
        StudentPet.vue
      gen/
        hq/
          pet/
            v1/
              pet_pb.ts
      stores/
        studentPet.js
      App.vue
      main.js
      router.js
      style.css
    package.json
    vite.config.js

  proto/
    hq/
      pet/
        v1/
          pet.proto
          pet.pb.go
          petv1connect/
            pet.connect.go

  deploy/
    docker-compose.yml
    postgres/
      init/
        001_create_assignment.sql

  web/
    built frontend assets
  ARCHITECTURE.md
  SERVER_PLAN.md
```

## Build Milestones

### Milestone 1: Project Foundation

- Create Go backend skeleton. Done.
- Create Vue 3 frontend. Done.
- Add Docker Compose for PostgreSQL using an official Postgres image. Done.
- Add database init SQL. Done.

### Milestone 2: API Foundation

- Add JSON API routes. Done.
- Add health check. Done.
- Add database connection. Done.

### Milestone 3: Authentication

- Add teacher login. Done.
- Add teacher cookie support. Done.
- Add teacher route checks. Done.
- Add real users and student authentication. Future.

### Milestone 4: Student Workflow

- Student entry from splash page. Done.
- One-at-a-time unanswered assignment view. Done.
- Text answer submission. Done.
- Grade summary and graded-question view. Done.

### Milestone 5: Teacher Workflow

- Teacher assignment creation. Done.
- Submitted answer review page. Done.
- Pass/fail and feedback form. Done.
- Reset submitted assignment while preserving attempt history. Done.
- Unified teacher questions workspace. Done.

### Milestone 6: Home-Network Mode

- Build the Vue frontend. Done.
- Serve the built frontend from Go. Done.
- Bind Go server to `0.0.0.0:8080`. Done.
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
- Use JSON over HTTP for homework workflows and Connect RPC for protobuf-backed pet workflows.
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
