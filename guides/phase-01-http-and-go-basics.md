# Phase 1 Manual — HTTP & Go Basics

**Duration:** 1–2 weeks  
**You need:** Go 1.22+, a terminal, `curl`  
**Audience:** First Go / first backend API — this manual is written for beginners

**How to use this document**

1. Read **Part A** (concepts) so the words make sense.
2. Follow **Part B** (build steps) in order. Each step gives **full file contents**, explains **why** each block exists, then asks you to **run and curl** before continuing.
3. Keep [The endpoint recipe](./the-endpoint-recipe.md) open — every step is that recipe in slow motion.
4. Do not skip ahead to “all endpoints.” Finish one step’s check before the next.

Replace `github.com/<you>/go-practice` with your real module path from `go.mod` everywhere you see it.

---

# Part A — Concepts (read first)

## A0. What you are building

**Desklog** is a work-log API. In Phase 1 it tracks **projects** and **tasks** in memory (RAM). When the server stops, data disappears. That is intentional.

By the end of this phase you will have:

- A Go program that listens on a port
- HTTP endpoints that accept and return JSON
- CRUD-style operations for projects and tasks
- Correct status codes and error responses
- A habit: **contract → model → store → handler → wire → verify**

Phases 2–6 add MongoDB, auth, reporting, and deployment on top of this habit — not a different one.

---

## A1. How a backend API works

A **backend API** waits for **HTTP requests** and sends **HTTP responses**.

| Request part | Example | Meaning |
|--------------|---------|---------|
| Method | `GET`, `POST`, `PATCH`, `DELETE` | Action |
| Path | `/projects/abc123` | Which resource |
| Headers | `Content-Type: application/json` | Metadata |
| Body | `{"name":"go-practice"}` | Payload (often on POST/PATCH) |

| Response part | Example | Meaning |
|---------------|---------|---------|
| Status code | `200`, `201`, `404` | Outcome |
| Headers | `Content-Type: application/json` | Metadata |
| Body | `{"id":"...","name":"..."}` | Payload |

The Go function that handles one route is a **handler**.

---

## A2. Modules, packages, folders

- A **module** (`go.mod`) is your project identity and import path.
- A **package** is a folder of `.go` files that all start with the same `package name`.
- **Every** `.go` file in a package must be valid. An empty `task.go` breaks the whole `handler` package.

| Path | Job |
|------|-----|
| `cmd/api/main.go` | Start server, create store, register routes only |
| `internal/model/` | Structs (`Project`, `Task`) — no HTTP |
| `internal/store/` | In-memory maps + mutex + CRUD methods |
| `internal/handler/` | HTTP: parse, validate, call store, write JSON |

`internal/` means other modules cannot import these packages.

**Do not** create `service/` or `repository/` yet — Phase 2.

---

## A3. Types you need

**Struct** — named fields:

```go
type Project struct {
	ID   string
	Name string
}
```

Capitalized fields are **exported** (usable from other packages).

**Slice** — list: `[]Project`  
**Map** — lookup by ID: `map[string]Project`  
**Error** — check `if err != nil` immediately. Go has no exceptions.

Pointers (`*http.Request`, `*MemoryStore`) mean “this value can be shared/modified.” You do not need deep pointer theory for Phase 1.

---

## A4. JSON

APIs speak JSON. Struct **tags** control field names:

```go
Name string `json:"name"`
```

| Tag | Effect |
|-----|--------|
| `json:"name"` | JSON key is `name` |
| `json:"description,omitempty"` | Omit if empty |
| `json:"created_at"` | Snake_case in JSON |

Encode to response: `json.NewEncoder(w).Encode(v)`  
Decode from body: `json.NewDecoder(r.Body).Decode(&req)`

Always set `Content-Type: application/json` before writing JSON.

---

## A5. Methods and status codes

| Method | Typical use |
|--------|-------------|
| GET | Read |
| POST | Create |
| PATCH | Partial update |
| DELETE | Remove |

| Code | When |
|------|------|
| 200 | Successful GET/PATCH |
| 201 | Successful POST |
| 204 | Successful DELETE (no body) |
| 400 | Bad JSON / validation |
| 404 | ID not found |
| 500 | Unexpected bug |

Never return `200` with `{"error":"..."}`. Use the real status code.

---

## A6. Handler pattern (memorize this)

```
1. Parse    — path params, JSON body
2. Validate — required fields, allowed values
3. Execute  — call store
4. Respond  — JSON + status
5. On error — {"error":"..."} + 4xx/5xx
```

That is steps 5–7 of [the recipe](./the-endpoint-recipe.md) in more detail.

---

# Part B — Build steps (do these in order)

Each step: **create/replace files → read the “why” → run check → only then continue.**

---

## Step 1 — Module and folders

**Goal:** Go knows your module; folders exist for later files.

```bash
cd /path/to/go-practice
go mod init github.com/<you>/go-practice
mkdir -p cmd/api internal/model internal/store internal/handler
```

**Why**

| Piece | Why |
|-------|-----|
| `go mod init` | Creates `go.mod` so imports like `github.com/<you>/go-practice/internal/handler` work |
| `cmd/api` | Convention for the runnable program (`go run ./cmd/api`) |
| `internal/...` | Private packages for model, store, handlers |

**Check:** `go.mod` exists and folders are present. No server yet.

---

## Step 2 — Health endpoint only

**Recipe for this step:** contract → handler → wire → verify (no model/store yet).

**Contract**

- `GET /health`
- Success: `200` + `{"status":"ok"}`
- No auth, no body

### File: `internal/handler/health.go` (full file)

```go
package handler

import (
	"fmt"
	"net/http"
)

// HealthHandler answers GET /health.
// It proves the process is alive. No store, no database.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `package handler` | All handler files share this package name |
| `HealthHandler` capitalized | So `main` can call `handler.HealthHandler` |
| Set `Content-Type` | Clients know the body is JSON |
| Fixed JSON string | Health needs no structs yet |
| **No** `func main` here | Server startup belongs only in `cmd/api` |

### File: `cmd/api/main.go` (full file for this step)

```go
package main

import (
	"log"
	"net/http"

	"github.com/<you>/go-practice/internal/handler"
)

func main() {
	// 1. Register routes BEFORE starting the server.
	http.HandleFunc("GET /health", handler.HealthHandler)

	// 2. ListenAndServe blocks forever. Nothing after this runs
	//    until the server shuts down — so do not put setup below it.
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `package main` + `func main` | Required for `go run ./cmd/api` |
| Import `internal/handler` | Uses your HealthHandler |
| `HandleFunc("GET /health", ...)` | Go 1.22+ method+path routing |
| Setup before `ListenAndServe` | Common beginner bug: code after Listen never runs |

**Also create stub packages so empty files do not break builds later.** Until you fill them, each must at least say:

```go
// internal/handler/task.go
package handler
```

```go
// internal/model/project.go and task.go — wait for Step 3, or:
package model
```

```go
// internal/store/memory.go — wait for Step 4, or:
package store
```

If a file is completely empty, Go reports `expected 'package', found 'EOF'`.

**Check**

```bash
go run ./cmd/api
# other terminal:
curl -i http://localhost:8080/health
```

Expect `200` and `{"status":"ok"}`. Stop the server with Ctrl+C when done.

---

## Step 3 — Models

**Recipe:** contract shapes → model files.

### File: `internal/model/project.go` (full file)

```go
package model

import "time"

// Project is a container for work (e.g. "go-practice").
// Defined here so handler and store share one type.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `type Project` not `type model.Project` | You are *inside* package `model`; other packages write `model.Project` when importing |
| JSON tags | API uses snake_case keys |
| `omitempty` on description | Omit empty description from JSON |
| `time.Time` | Timestamps; we will set them in UTC in the handler |

### File: `internal/model/task.go` (full file)

```go
package model

import "time"

// Task is a unit of work inside a project.
type Task struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // todo | doing | done
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `ProjectID` | Links task to project; set from URL later, not trusted from body alone |
| `Status` string | Simple enum for Phase 1; validate in the handler |

**Check:** files save with no red errors. Still no need to run the server for models alone.

---

## Step 4 — In-memory store

**Recipe:** data layer. Handlers must not own maps.

Maps need `make(...)` before write, or you panic. `NewMemoryStore` does that. A **mutex** prevents crashes when two requests touch the map at once (HTTP is concurrent).

### File: `internal/store/memory.go` (full file for projects + tasks)

```go
package store

import (
	"sync"

	"github.com/<you>/go-practice/internal/model"
)

// MemoryStore holds all Phase 1 data in RAM.
// Restarting the process clears everything — expected for this phase.
type MemoryStore struct {
	mu       sync.Mutex
	projects map[string]model.Project
	tasks    map[string]model.Task
}

// NewMemoryStore constructs an empty store with initialized maps.
// Call this once from main and pass the pointer into handlers.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects: make(map[string]model.Project),
		tasks:    make(map[string]model.Task),
	}
}

func (s *MemoryStore) CreateProject(p model.Project) model.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
	return p
}

func (s *MemoryStore) ListProjects() []model.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Project, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, p)
	}
	return out
}

func (s *MemoryStore) GetProject(id string) (model.Project, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	return p, ok
}

func (s *MemoryStore) UpdateProject(p model.Project) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[p.ID]; !ok {
		return false
	}
	s.projects[p.ID] = p
	return true
}

func (s *MemoryStore) DeleteProject(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return false
	}
	delete(s.projects, id)
	for tid, t := range s.tasks {
		if t.ProjectID == id {
			delete(s.tasks, tid)
		}
	}
	return true
}

func (s *MemoryStore) CreateTask(t model.Task) model.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return t
}

func (s *MemoryStore) ListTasksByProject(projectID string) []model.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Task, 0)
	for _, t := range s.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out
}

func (s *MemoryStore) GetTask(id string) (model.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *MemoryStore) UpdateTask(t model.Task) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return false
	}
	s.tasks[t.ID] = t
	return true
}

func (s *MemoryStore) DeleteTask(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `model.Project` / `model.Task` | Types live in `model`; store imports them |
| `NewMemoryStore` + `make` | Nil maps panic on assign |
| `sync.Mutex` + `Lock`/`Unlock` | Safe under concurrent HTTP |
| `defer Unlock` | Unlock even if function returns early |
| `Get*` returns `(value, bool)` | Caller maps `false` → HTTP 404 |
| `DeleteProject` removes tasks | Avoid orphan tasks when project is deleted |

**Check:** file compiles conceptually; you will compile for real in Step 5 with handlers.

---

## Step 5 — POST /projects (first write endpoint)

**Recipe:** contract → (model/store done) → handler → wire → verify.

**Contract**

- `POST /projects`
- Body: `{"name":"...","description":"..."}` (`name` required)
- Success: `201` + full project including `id` and timestamps
- Errors: `400` invalid JSON or missing name

### File: `internal/handler/project.go` (start with helpers + create only)

```go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/store"
)

// writeError sends a consistent error JSON body.
// Use this for every 4xx/5xx so clients always see {"error":"..."}.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func newID() string {
	// Simple unique-enough ID for Phase 1 (no extra dependency).
	// Phase 2 will use MongoDB ObjectIDs instead.
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CreateProjectHandler handles POST /projects.
// Parameter name is "mem" so it does not shadow the imported store package.
func CreateProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		// 2. Validate
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}

		// 3. Execute
		now := time.Now().UTC()
		project := model.Project{
			ID:          newID(),
			Name:        req.Name,
			Description: req.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		project = mem.CreateProject(project)

		// 4. Respond
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // 201
		_ = json.NewEncoder(w).Encode(project)
	}
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `writeError` | One error shape for all endpoints |
| `CreateProjectHandler(mem *store.MemoryStore) http.HandlerFunc` | Returns a handler that *closes over* the store — `main` creates one store and passes it in |
| Decode into anonymous `req` struct | Request body may not match full `Project` (no client-supplied `id`) |
| Validate `name` | Business rule: name required → 400 |
| `newID()` + UTC times | Server assigns identity and timestamps |
| `201` + encode project | Matches create contract |
| Param named `mem` | Avoids `store` colliding with package name `store` |

### Update `cmd/api/main.go` (full file)

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/<you>/go-practice/internal/handler"
	"github.com/<you>/go-practice/internal/store"
)

func main() {
	mem := store.NewMemoryStore()

	http.HandleFunc("GET /health", handler.HealthHandler)
	http.HandleFunc("POST /projects", handler.CreateProjectHandler(mem))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| `mem := store.NewMemoryStore()` first | One shared store for all handlers |
| Pass `mem` into `CreateProjectHandler` | Dependency injection without a framework |
| `PORT` env | Phase 6 deploy can change port without code edits |
| All `HandleFunc` before Listen | Wiring complete before accepting traffic |

**Check**

```bash
go run ./cmd/api

curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"go-practice","description":"Phase 1"}'

curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d 'not json'
```

Expect `201` with an `id`, then `400` for bad JSON.

---

## Step 6 — GET /projects and GET /projects/{id}

**Contracts**

| Endpoint | Success | Errors |
|----------|---------|--------|
| `GET /projects` | `200` + JSON array (empty list = `[]`, not 404) | — |
| `GET /projects/{id}` | `200` + one project | `404` if missing |

### Add to `internal/handler/project.go`

```go
// ListProjectsHandler handles GET /projects.
func ListProjectsHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects := mem.ListProjects()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projects) // 200 by default
	}
}

// GetProjectHandler handles GET /projects/{id}.
func GetProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id") // from "{id}" in the route pattern
		project, ok := mem.GetProject(id)
		if !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(project)
	}
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| No JSON decode on GET list | Nothing to parse from body |
| Encode slice even if empty | `[]` is correct; 404 would mean “route missing,” not “no data” |
| `r.PathValue("id")` | Reads `{id}` from `GET /projects/{id}` |
| `ok == false` → 404 | Store signals missing; handler maps to HTTP |

### Register in `main.go`

```go
http.HandleFunc("GET /projects", handler.ListProjectsHandler(mem))
http.HandleFunc("GET /projects/{id}", handler.GetProjectHandler(mem))
```

**Check**

```bash
# create, copy id from response, then:
curl -s http://localhost:8080/projects
curl -s http://localhost:8080/projects/<ID>
curl -i http://localhost:8080/projects/does-not-exist
```

---

## Step 7 — Task endpoints

**Contracts**

| Endpoint | Behavior |
|----------|----------|
| `GET /projects/{id}/tasks` | 404 if project missing; else `200` + task array |
| `POST /projects/{id}/tasks` | Body `{"title":"...","status":"todo"}`; 404 if project missing; 400 if bad input; 201 + task |
| `GET /tasks/{id}` | 200 or 404 |

Rules for create:

1. Project must exist  
2. Set `project_id` from path (never trust body for ownership)  
3. Default status to `"todo"` if empty  
4. If status provided, only allow `todo`, `doing`, `done`

### File: `internal/handler/task.go` (full file)

```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/store"
)

func validStatus(s string) bool {
	return s == "todo" || s == "doing" || s == "done"
}

// ListTasksByProjectHandler handles GET /projects/{id}/tasks.
func ListTasksByProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if _, ok := mem.GetProject(projectID); !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		tasks := mem.ListTasksByProject(projectID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	}
}

// CreateTaskHandler handles POST /projects/{id}/tasks.
func CreateTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if _, ok := mem.GetProject(projectID); !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var req struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.Status == "" {
			req.Status = "todo"
		}
		if !validStatus(req.Status) {
			writeError(w, http.StatusBadRequest, "status must be todo, doing, or done")
			return
		}

		now := time.Now().UTC()
		task := model.Task{
			ID:        newID(),
			ProjectID: projectID, // from path, not body
			Title:     req.Title,
			Status:    req.Status,
			CreatedAt: now,
			UpdatedAt: now,
		}
		task = mem.CreateTask(task)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	}
}

// GetTaskHandler handles GET /tasks/{id}.
func GetTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		task, ok := mem.GetTask(id)
		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}
}
```

**Why this block**

| Line / idea | Why |
|-------------|-----|
| Check project before list/create | Nested resource: parent must exist |
| `ProjectID: projectID` from path | Client cannot attach a task to someone else’s project ID via body |
| Default + validate status | Keeps data clean |
| Reuse `writeError` / `newID` | Same package `handler` — shared helpers |

### Register in `main.go`

```go
http.HandleFunc("GET /projects/{id}/tasks", handler.ListTasksByProjectHandler(mem))
http.HandleFunc("POST /projects/{id}/tasks", handler.CreateTaskHandler(mem))
http.HandleFunc("GET /tasks/{id}", handler.GetTaskHandler(mem))
```

**Check** — use a real project ID from Step 5:

```bash
curl -i -X POST http://localhost:8080/projects/<PROJECT_ID>/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"First task"}'

curl -s http://localhost:8080/projects/<PROJECT_ID>/tasks
curl -s http://localhost:8080/tasks/<TASK_ID>
```

---

## Step 8 — Recipe: adding any new endpoint

You now have enough code to **transfer**. For any new endpoint (including stretch below), fill this out, then code in that order:

```text
1. Contract:  METHOD /path
   Request:   ...
   Success:   STATUS + body
   Errors:    ...

2. Model:     new fields? → internal/model/

3. Store:     new method? → internal/store/memory.go

4. Rules:     what must be true? (in handler for Phase 1)

5. Handler:   VerbResourceHandler in handler/<resource>.go
              parse → validate → store → respond

6. Wire:      http.HandleFunc("METHOD /path", handler.Xxx(mem))
              in cmd/api/main.go BEFORE ListenAndServe

7. Verify:    curl -i ...
```

That is the same process professionals use. Later phases only swap **store → repository + service**; the order stays.

---

## Step 9 — Stretch (recommended before Phase 2)

Use the recipe above. Store methods `UpdateProject`, `DeleteProject`, `UpdateTask`, `DeleteTask` are already in Step 4.

| Endpoint | Notes |
|----------|--------|
| `PATCH /projects/{id}` | Update name/description; bump `updated_at`; 404 if missing |
| `DELETE /projects/{id}` | Delete project + its tasks; `204` or `200` |
| `PATCH /tasks/{id}` | Update title/status |
| `DELETE /tasks/{id}` | Remove task |

Optional logging middleware (wrap a handler; register in `main`):

```go
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// http.HandleFunc("GET /health", withLogging(handler.HealthHandler))
```

---

## Step 10 — Full curl smoke test

```bash
go run ./cmd/api

curl -i http://localhost:8080/health

curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"go-practice","description":"Phase 1"}'

curl -s http://localhost:8080/projects
curl -s http://localhost:8080/projects/<ID>

curl -i -X POST http://localhost:8080/projects/<PROJECT_ID>/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"First task"}'

curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d 'not json'

curl -i http://localhost:8080/projects/does-not-exist
```

Race detector (optional):

```bash
go run -race ./cmd/api
```

---

## Step 11 — README for this phase

Document in `README.md`:

1. What Desklog is (one sentence)  
2. Prerequisites (Go version)  
3. How to run: `go run ./cmd/api`  
4. Default port  
5. List of endpoints  
6. One example curl for create + list  

---

## Common mistakes

| Mistake | Why wrong | Fix |
|---------|-----------|-----|
| Empty `.go` file in a package | Build fails with EOF | At least `package name` |
| `func main` inside `handler/` | Wrong package; two programs | Only `cmd/api/main.go` |
| Code after `ListenAndServe` | Never runs | Wire store/routes above it |
| `type model.Project struct` inside `package model` | Invalid syntax | `type Project struct` |
| `"uuid"` import | Not a real module path | Use `newID()` or `github.com/google/uuid` |
| Param named `store` + import `store` | Confusing / shadows | Call it `mem` |
| Copy-paste create into list handler | List must not decode a body | Call `ListProjects` only |
| `200` on errors | Clients cannot tell failure | Use 4xx + `writeError` |
| Map without mutex | Data races | Always Lock around map access |

---

## Exit checklist

- [ ] You followed Steps 1–7 (stretch optional)  
- [ ] `go run ./cmd/api` works  
- [ ] Health, projects, and tasks match the contracts  
- [ ] Errors return `{"error":"..."}` with correct status  
- [ ] Mutex protects maps  
- [ ] README documents how to run and test  
- [ ] You can explain: **request → handler → store → response**  
- [ ] You can add a new endpoint using Step 8 without being told which file  

**Commit:** `feat(phase-1): in-memory projects and tasks API`

---

**Next:** [Phase 2 Manual — MongoDB & Structure](./phase-02-mongodb-and-structure.md)  
**Always:** [The endpoint recipe](./the-endpoint-recipe.md)
